# Claude Configuration

This file contains configuration and notes for Claude Code.

## Project Information

- **Working Directory**: /home/satoru314/programf/storage
- **Git Repository**: Yes
- **Current Branch**: develop
- **Main Branch**: main

## Commands

Add frequently used commands here for reference:

```bash
# Build
# Add your build command here

# Test  
# Add your test command here

# Lint
# Add your lint command here

# Type Check
# Add your type check command here
```

---

# S3直アップロード画像管理システム設計

## 目的と前提

* **目的**: フロント（React/Vite, TypeScript）から**S3直プリサイン**を用いて画像をアップロード／閲覧できる最小アプリをローカルで動かす。
* **構成**: SPA（localhost:5173）／API（Go Echo, localhost:8080）／DB（PostgreSQL, Docker）／S3（AWS）
* **モノレポ構成**: フロントとバックエンドを同一リポジトリで管理
* **認証**: なし（誰がアクセスしても同じ結果）。
* **ストレージ方針**: S3には**画像本体のみ**格納。メタデータはDBに保存。
* **アップロード方式**: フロント→S3へ**直接PUT**（バックエンドが**presigned PUT URL**を発行）。
* **閲覧方式**: S3の**presigned GET URL**をバックエンドが発行。
* **想定最大サイズ**: 単発PUTで扱えるサイズ（例: 50MB）。マルチパートは今回範囲外。

## アーキテクチャ（ローカル）

```
[React/Vite (5173)]  --(POST /images/upload-request)-->  [Go/Echo (8080)]
       |                                                        |
       |                                        [Gorm] <-> [PostgreSQL]
       |                                                        |
       |<- (presigned PUT URL) ---------------------------------|
       |
[ブラウザからS3へPUT]  -->  [AWS S3 バケット: app-images]
       |
       |--(POST /images/upload-complete)--> [Go/Echo] → DB更新
       |
  表示: (POST /images/view-urls or /{id}/view-url)
       |<- presigned GET URL -- [Go/Echo]
       |-- GET ---------------> [AWS S3]
```

## 環境変数・ポート

* フロント: `http://localhost:5173`
* バック: `http://localhost:8080`
* AWS S3: 本番環境のS3バケット
* Postgres: `localhost:5432`

**API（Go）用環境変数例**

```
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/images?sslmode=disable
AWS_REGION=ap-northeast-1
S3_BUCKET=app-images
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
PRESIGN_PUT_TTL_SEC=300
PRESIGN_GET_TTL_SEC=300
ALLOWED_ORIGIN=http://localhost:5173
```

## S3設定

### バケット

* 名称: `app-images`
* 既定は**PRIVATE**（匿名公開はしない）
* リージョン: `ap-northeast-1`（東京リージョン）

### CORS（S3バケットのCORS設定JSON）

```json
[
  {
    "AllowedOrigin": ["http://localhost:5173"],
    "AllowedMethod": ["GET", "PUT", "HEAD"],
    "AllowedHeader": [
      "*",
      "Content-Type",
      "Content-MD5",
      "x-amz-acl",
      "x-amz-checksum-sha256",
      "x-amz-meta-*"
    ],
    "ExposeHeader": ["ETag", "x-amz-version-id"],
    "MaxAgeSeconds": 3000
  }
]
```

> 備考: フロントからのS3直PUT/GETが通るように、`AllowedOrigin`はViteのオリジンに合わせる。AWS S3コンソールのバケット設定で適用。

## データベース

### スキーマ（最小）

```sql
CREATE TABLE IF NOT EXISTS images (
  id               UUID PRIMARY KEY,
  object_key       TEXT NOT NULL UNIQUE,
  original_name    TEXT NOT NULL,
  mime_type        TEXT NOT NULL,
  byte_size        BIGINT NOT NULL,
  width            INT NULL,
  height           INT NULL,
  etag             TEXT NULL,
  storage_status   TEXT NOT NULL, -- 'requested' | 'uploaded' | 'failed' | 'deleted'
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  uploaded_at      TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_images_created ON images(created_at DESC);
```

### Gormモデル例（型定義だけ）

```go
 type Image struct {
   ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
   ObjectKey     string    `gorm:"uniqueIndex;not null"`
   OriginalName  string    `gorm:"not null"`
   MimeType      string    `gorm:"not null"`
   ByteSize      int64     `gorm:"not null"`
   Width         *int
   Height        *int
   ETag          *string
   StorageStatus string    `gorm:"not null"`
   CreatedAt     time.Time
   UploadedAt    *time.Time
 }
```

## バックエンドAPI（Echo）

### 1) ヘルスチェック

* `GET /healthz`
* **200 OK** `{ "status": "ok" }`

### 2) アップロード用URL生成（presigned PUT）

* `POST /images/upload-request`

#### Request (JSON)

```json
{
  "fileName": "cat.jpg",
  "contentType": "image/jpeg",
  "fileSize": 4238121
}
```

#### Validation

* `contentType` は `image/jpeg|image/png|image/webp|image/heic` のみ
* `fileSize` ≤ 50MB（環境変数で可変）
* `fileName` に制御文字やパストラバーサルがないこと

#### Server Steps

1. `object_key` を生成: `y=YYYY/m=MM/d=DD/<uuid>_<safeBaseName>.<ext>`
2. DBに仮レコードを保存（`storage_status='requested'`）
3. S3 **presigned PUT URL** を発行（TTL=`PRESIGN_PUT_TTL_SEC`）
4. 必要ヘッダ（`Content-Type` など）を返す

#### Response (JSON)

```json
{
  "image": {
    "id": "1f4a5b98-...",
    "objectKey": "y=2025/m=08/d=24/1f4a..._cat.jpg",
    "originalName": "cat.jpg",
    "mimeType": "image/jpeg",
    "byteSize": 4238121,
    "status": "requested"
  },
  "upload": {
    "method": "PUT",
    "url": "https://app-images.s3.ap-northeast-1.amazonaws.com/y=2025/.../cat.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&...",
    "headers": {
      "Content-Type": "image/jpeg"
    },
    "expiresInSec": 300
  }
}
```

### 3) アップロード完了通知

* `POST /images/upload-complete`

#### Request (JSON)

```json
{
  "id": "1f4a5b98-...",
  "objectKey": "y=2025/m=08/d=24/1f4a..._cat.jpg"
}
```

#### Response

```json
{ "status": "ok" }
```

### 4) 画像一覧（キーセットページング）

* `GET /images?limit=20&cursor=...&sort=created_at_desc`

#### Response

```json
{
  "items": [
    {
      "id": "1f4a5b98-...",
      "objectKey": "y=2025/m=08/d=24/1f4a..._cat.jpg",
      "originalName": "cat.jpg",
      "mimeType": "image/jpeg",
      "byteSize": 4238121,
      "uploadedAt": "2025-08-24T12:34:56Z"
    }
  ],
  "nextCursor": "eyJjcmVhdGVkQXQiOiIyMDI1LTA4LTI0VDEyOjM0OjU2WiIsImlkIjoiMWY0YS..."
}
```

### 5) 単一画像メタ取得

* `GET /images/{id}`

### 6) 表示用URLの一括発行（presigned GET, N+1対策）

* `POST /images/view-urls`

#### Request

```json
{
  "requests": [
    { "id": "1f4a5b98-..." },
    { "id": "7b39c1f2-..." }
  ],
  "ttlSec": 300
}
```

#### Response

```json
{
  "results": [
    {
      "id": "1f4a5b98-...",
      "url": "https://app-images.s3.ap-northeast-1.amazonaws.com/y=2025/.../cat.jpg?X-Amz-...",
      "expiresAt": "2025-08-24T12:40:56Z"
    }
  ]
}
```

### 7) 単体の表示用URL（任意）

* `POST /images/{id}/view-url`

### エラーレスポンス共通

```json
{
  "error": {
    "code": "BadRequest",
    "message": "contentType not allowed",
    "requestId": "a1b2c3d4"
  }
}
```

## フロント（React/Vite, TS）

### 主要フロー

#### アップロード

1. ファイル選択 → `POST /images/upload-request`
2. 返却された `upload.url` に **PUT**（ヘッダ `Content-Type` をセット）
3. 成功したら `POST /images/upload-complete`
4. 完了トースト表示

#### 一覧表示

1. `GET /images?limit=30` でメタ一覧取得
2. `POST /images/view-urls` に id配列を渡し、表示用の **GET presign URL** を一括取得
3. `<img src={url}>` で表示（`loading="lazy"` 推奨）

### 型（API I/F）

```ts
export type UploadRequestReq = {
  fileName: string;
  contentType: string;
  fileSize: number;
};

export type UploadRequestRes = {
  image: {
    id: string;
    objectKey: string;
    originalName: string;
    mimeType: string;
    byteSize: number;
    status: 'requested' | 'uploaded' | 'failed' | 'deleted';
  };
  upload: {
    method: 'PUT';
    url: string;
    headers: Record<string, string>;
    expiresInSec: number;
  };
};

export type UploadCompleteReq = {
  id: string;
  objectKey: string;
};

export type ImageMeta = {
  id: string;
  objectKey: string;
  originalName: string;
  mimeType: string;
  byteSize: number;
  uploadedAt?: string;
};

export type ViewUrlsReq = {
  requests: { id: string }[];
  ttlSec?: number;
};
export type ViewUrlsRes = {
  results: { id: string; url: string; expiresAt: string }[];
};
```

## バックエンド実装ノート（Echo）

* ルーティング

  * `GET /healthz`
  * `POST /images/upload-request`
  * `POST /images/upload-complete`
  * `GET /images`
  * `GET /images/:id`
  * `POST /images/view-urls`

* CORS（Echoのミドルウェア）

  * `AllowOrigins: [ALLOWED_ORIGIN]`
  * `AllowMethods: GET,POST,OPTIONS`
  * `AllowHeaders: Content-Type, Authorization`

* S3クライアント

  * Go AWS SDK v2 の標準設定
  * リージョン設定: `aws.Config{Region: AWS_REGION}`

* presign 生成

  * PUT: `s3.NewPresignClient().PresignPutObject`
  * GET: `s3.NewPresignClient().PresignGetObject`

* object key の例

  * `y=2025/m=08/d=24/<uuid>_<safeBaseName>.<ext>`

## Docker・AWS設定

* **PostgreSQL**: `postgres:15`

  * ユーザー/パス: `postgres/postgres`
  * DB: `images`

* **AWS S3**:

  * バケット作成: AWS CLIまたはコンソールで `app-images` バケット作成
  * CORS設定: S3コンソールまたはAWS CLIでCORS設定を適用
  * IAM設定: S3アクセス権限を持つユーザーまたはロールの設定

## モノレポ構成

```
/
├── api/          # Go Echo バックエンド
├── app/          # React/Vite フロントエンド
├── docker-compose.yml  # PostgreSQL用
├── README.md
└── claude.md
```

## シーケンス（テキスト）

### アップロード

```
SPA -> API: POST /images/upload-request {fileName, contentType, fileSize}
API -> DB: INSERT images(status=requested)
API -> S3: presign PUT (TTL=300)
API -> SPA: {image, upload{url, headers}}
SPA -> S3: PUT file (headers.Content-Type)
S3 -> SPA: 200 OK (ETag)
SPA -> API: POST /images/upload-complete {id, objectKey}
API -> S3: HeadObject
API -> DB: UPDATE images SET status=uploaded, uploaded_at=now, etag=...
API -> SPA: {status: ok}
```

### 一覧＋表示

```
SPA -> API: GET /images?limit=30
API -> DB: SELECT ... ORDER BY created_at DESC LIMIT 30
API -> SPA: {items, nextCursor}
SPA -> API: POST /images/view-urls {requests:[{id:...}, ...]}
API -> DB: SELECT object_key FROM images WHERE id IN (...)
API -> S3: presign GET (TTL=300) for each
API -> SPA: {results:[{id,url,expiresAt}, ...]}
SPA -> AWS S3: GET {url}
```

## テスト観点（簡易）

* 画像: JPEG/PNG/WebP/HEIC を各1枚アップ→S3に存在／DBが `uploaded` になる
* PUT失敗（MIME不一致/サイズ超過）→ 400
* 未完了の `upload-complete`（存在しないkey）→ 409 or 404
* 一覧: 登録順に降順で返る、`limit`/`cursor` でページングできる
* 表示URL: 期限切れでアクセス不可になる（TTLを短くして検証）

## まとめ

* **フロント**は `/images/upload-request` → **S3 PUT** → `/images/upload-complete`
* **一覧**は `GET /images` → `POST /images/view-urls`
* すべて**認証なし**で同一結果（ローカル開発向け）
* 本資料は **エンドポイントとリクエスト内容**に重点化済み。実装はこのI/Fに沿って進める。

---

# 開発タスク一覧

## 実装順序

1. **プロジェクト構造とディレクトリを作成**
   - モノレポ構成確認（app/フロント, api/バック）
   - 必要なファイル作成

2. **Docker環境（PostgreSQL）をセットアップ**  
   - docker-compose.yml作成
   - PostgreSQL接続設定

3. **Go Echo APIの基盤を構築**
   - ヘルスチェック、CORS、DB接続の基本構成

4. **データベーススキーマとGormモデルを実装**
   - imagesテーブル作成
   - Gormモデル定義

5. **S3クライアント設定と環境変数管理を実装**
   - AWS SDK設定
   - 環境変数管理

6. **画像アップロード関連APIエンドポイントを実装**
   - POST /images/upload-request
   - POST /images/upload-complete

7. **画像一覧・取得APIエンドポイントを実装**
   - GET /images（ページング対応）
   - GET /images/:id

8. **表示用URL発行APIエンドポイントを実装**
   - POST /images/view-urls（N+1対策）

9. **React/Vite フロントエンドの基盤をセットアップ**
   - Vite+TypeScript環境構築
   - ルーティング設定

10. **フロントエンドのAPI型定義とHTTPクライアントを実装**
    - TypeScript型定義
    - APIクライアント実装

11. **画像アップロード機能のUIとロジックを実装**
    - ファイル選択→S3直PUT→完了通知のフロー

12. **画像一覧表示機能のUIとロジックを実装** 
    - 画像グリッド表示
    - presigned URL取得・表示

13. **AWS S3バケット作成とCORS設定**
    - S3バケット準備
    - CORS設定

14. **エラーハンドリングとバリデーションの実装**
    - バリデーション強化
    - 例外処理実装

15. **動作確認とテスト**
    - 統合テスト
    - 動作検証