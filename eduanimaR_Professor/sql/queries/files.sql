-- sql/queries/files.sql

-- name: GetFileByID :one
SELECT *
FROM files
WHERE file_id = $1;

-- name: GetFileByIDAndUserID :one
SELECT *
FROM files
WHERE file_id = $1
  AND user_id = $2;

-- name: ListFilesBySubjectID :many
SELECT *
FROM files
WHERE subject_id = $1
ORDER BY uploaded_at DESC;

-- name: CreateFile :one
INSERT INTO files (
    file_id,
    subject_id,
    user_id,
    name,
    storage_path,
    mime_type,
    size_bytes,
    status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateFileStatus :one
UPDATE files
SET
    status        = $2,
    error_message = $3,
    processed_at  = CASE
                        WHEN $2::file_status = 'ready' THEN NOW()
                        ELSE processed_at
                    END
WHERE file_id = $1
RETURNING *;

-- name: DeleteFile :exec
DELETE FROM files
WHERE file_id = $1
  AND user_id = $2;
