-- sql/queries/chunks.sql

-- name: ListChunksByFileID :many
SELECT
  m.id AS chunk_id,
  m.raw_file_id AS file_id,
  rf.subject_id,
  m.page_start AS page_number,
  m.sequence_in_file AS chunk_index,
  m.content_markdown AS content,
  m.embedding,
  m.created_at
FROM materials m
JOIN raw_files rf ON rf.id = m.raw_file_id
WHERE m.raw_file_id = $1
  AND m.is_active = TRUE
ORDER BY m.sequence_in_file;

-- name: InsertChunk :one
INSERT INTO materials (
  id,
  raw_file_id,
  sequence_in_file,
  page_start,
  page_end,
  content_markdown,
  char_count,
  embedding
)
VALUES (
  sqlc.arg(chunk_id),
  sqlc.arg(file_id),
  sqlc.arg(chunk_index),
  sqlc.narg(page_number),
  sqlc.narg(page_number),
  sqlc.arg(content),
  char_length(sqlc.arg(content)),
  sqlc.arg(embedding)
)
RETURNING
  id AS chunk_id,
  raw_file_id AS file_id,
  (SELECT subject_id FROM raw_files WHERE id = raw_file_id) AS subject_id,
  page_start AS page_number,
  sequence_in_file AS chunk_index,
  content_markdown AS content,
  embedding,
  created_at;

-- name: SearchChunksByVector :many
SELECT
  m.id AS chunk_id,
  m.raw_file_id AS file_id,
  rf.subject_id,
  rf.original_filename AS file_name,
  m.page_start AS page_number,
  m.sequence_in_file AS chunk_index,
  m.content_markdown AS content,
  m.created_at
FROM materials m
JOIN raw_files rf ON rf.id = m.raw_file_id
WHERE rf.subject_id = $2
  AND rf.is_active = TRUE
  AND m.is_active = TRUE
ORDER BY m.embedding <=> $1::vector
LIMIT $3;

-- name: SearchChunksByText :many
SELECT
  m.id AS chunk_id,
  m.raw_file_id AS file_id,
  rf.subject_id,
  rf.original_filename AS file_name,
  m.page_start AS page_number,
  m.sequence_in_file AS chunk_index,
  m.content_markdown AS content,
  m.created_at
FROM materials m
JOIN raw_files rf ON rf.id = m.raw_file_id
WHERE rf.subject_id = $2
  AND rf.is_active = TRUE
  AND m.is_active = TRUE
  AND to_tsvector('simple', m.content_markdown) @@ plainto_tsquery('simple', $1)
ORDER BY ts_rank(to_tsvector('simple', m.content_markdown), plainto_tsquery('simple', $1)) DESC
LIMIT $3;

-- name: DeleteChunksByFileID :exec
UPDATE materials
SET is_active = FALSE,
    deleted_at = NOW(),
    updated_at = NOW()
WHERE raw_file_id = $1
  AND is_active = TRUE;
