-- sql/queries/chunks.sql

-- name: ListChunksByFileID :many
SELECT *
FROM chunks
WHERE file_id = $1
ORDER BY chunk_index;

-- name: InsertChunk :one
INSERT INTO chunks (
    chunk_id,
    file_id,
    subject_id,
    page_number,
    chunk_index,
    content,
    embedding
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: SearchChunksByVector :many
-- コサイン類似度でのベクトル検索（HNSW インデックス使用）
-- $1: query_embedding (vector), $2: subject_id, $3: limit
SELECT
    chunk_id,
    file_id,
    subject_id,
    page_number,
    chunk_index,
    content,
    created_at
FROM chunks
WHERE subject_id = $2
ORDER BY embedding <=> $1::vector
LIMIT $3;

-- name: SearchChunksByText :many
-- 全文検索（simple 辞書 / plainto_tsquery）
-- $1: query_text, $2: subject_id, $3: limit
SELECT
    chunk_id,
    file_id,
    subject_id,
    page_number,
    chunk_index,
    content,
    created_at
FROM chunks
WHERE subject_id = $2
  AND to_tsvector('simple', content) @@ plainto_tsquery('simple', $1)
ORDER BY ts_rank(to_tsvector('simple', content), plainto_tsquery('simple', $1)) DESC
LIMIT $3;

-- name: DeleteChunksByFileID :exec
DELETE FROM chunks
WHERE file_id = $1;
