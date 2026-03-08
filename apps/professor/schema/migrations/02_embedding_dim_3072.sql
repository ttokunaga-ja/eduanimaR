-- Migration 02: text-embedding-004 (768次元) → gemini-embedding-001 (3072次元) へ移行
-- text-embedding-004 が Google API から削除されたため gemini-embedding-001 を使用する。
-- pgvector の HNSW/IVFFlat は 2000次元上限のため、3072次元ではシーケンシャルスキャンを使用。

-- 既存の HNSW インデックスを削除（3072次元では使用不可）
DROP INDEX IF EXISTS idx_chunks_embedding_hnsw;

-- embedding カラムの次元数を 768 → 3072 に変更
ALTER TABLE chunks ALTER COLUMN embedding TYPE vector(3072);

-- NOTE: 3072次元は HNSW/IVFFlat の上限（2000次元）を超えるため ANN インデックスは作成しない。
-- 開発環境では順次スキャン（exact nearest neighbor）で代用する。
-- 本番環境では別の embedding モデル（768次元以下）の使用を検討すること。
