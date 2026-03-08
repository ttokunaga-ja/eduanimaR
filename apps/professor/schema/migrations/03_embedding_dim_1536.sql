-- Migration 03: embedding 次元数 3072 → 1536 + HNSW インデックス再作成
--
-- 背景:
--   gemini-embedding-001 は MRL（Matryoshka Representation Learning）に対応し、
--   128〜3072 次元を動的に指定できる。
--   pgvector 0.8.1 の HNSW インデックス上限が 2000 次元のため、
--   3072 次元（Migration 02）では HNSW を使用できずシーケンシャルスキャンになる。
--   1536 次元は精度を保ちつつ HNSW インデックスが利用可能なバランスの良い次元数。
--
-- 注意:
--   次元数変更後は既存チャンクの再インジェストが必要。
--   実行前に以下のコマンドで既存チャンクを削除すること:
--     TRUNCATE TABLE chunks;

-- Step 1: 既存の HNSW インデックスを削除（存在する場合）
DROP INDEX IF EXISTS idx_chunks_embedding_hnsw;

-- Step 2: embedding カラムの次元数を変更 (3072 → 1536)
ALTER TABLE chunks ALTER COLUMN embedding TYPE vector(1536);

-- Step 3: HNSW インデックスを再作成
--   1536 次元は pgvector HNSW 上限（2000 次元）以内のため利用可能
--   m=16, ef_construction=64 は標準的なバランス設定
CREATE INDEX idx_chunks_embedding_hnsw ON chunks
  USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);
