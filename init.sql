-- 初期化用SQLスクリプト
-- imagesテーブルを作成

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS images (
  id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  object_key       TEXT NOT NULL UNIQUE,
  original_name    TEXT NOT NULL,
  mime_type        TEXT NOT NULL,
  byte_size        BIGINT NOT NULL,
  width            INT NULL,
  height           INT NULL,
  etag             TEXT NULL,
  storage_status   TEXT NOT NULL CHECK (storage_status IN ('requested', 'uploaded', 'failed', 'deleted')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  uploaded_at      TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_images_created ON images(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_images_status ON images(storage_status);

-- テストデータ（オプション）
-- INSERT INTO images (object_key, original_name, mime_type, byte_size, storage_status) 
-- VALUES ('test/sample.jpg', 'sample.jpg', 'image/jpeg', 1024, 'uploaded');