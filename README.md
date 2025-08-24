# S3直アップロード画像管理システム

フロント（React/Vite, TypeScript）からS3直プリサインを用いて画像をアップロード／閲覧できるアプリケーション。

## 構成

- **フロントエンド**: `app/` - React + Vite + TypeScript
- **バックエンド**: `api/` - Go + Echo framework
- **データベース**: PostgreSQL（Docker）
- **ストレージ**: AWS S3

## 開発環境

- フロントエンド: http://localhost:5173
- バックエンド: http://localhost:8080
- PostgreSQL: localhost:5432

## セットアップ

### 1. PostgreSQL起動

```bash
docker-compose up -d
```

### 2. バックエンド起動

```bash
cd api
go run cmd/main.go
```

### 3. フロントエンド起動

```bash
cd app
npm run dev
```

## 環境変数

バックエンド用の環境変数は `.env` ファイルまたは環境変数で設定：

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

## API仕様

詳細は`claude.md`を参照。