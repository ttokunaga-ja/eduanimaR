-- sql/queries/files.sql

-- name: GetFileByID :one
SELECT
  id AS file_id,
  subject_id,
  user_id,
  original_filename AS name,
  gcs_object_path AS storage_path,
  mime_type,
  file_size_bytes AS size_bytes,
  status,
  NULL::text AS error_message,
  created_at AS uploaded_at,
  processed_at
FROM raw_files
WHERE id = $1
  AND is_active = TRUE;

-- name: GetFileByIDAndUserID :one
SELECT
  id AS file_id,
  subject_id,
  user_id,
  original_filename AS name,
  gcs_object_path AS storage_path,
  mime_type,
  file_size_bytes AS size_bytes,
  status,
  NULL::text AS error_message,
  created_at AS uploaded_at,
  processed_at
FROM raw_files
WHERE id = $1
  AND user_id = $2
  AND is_active = TRUE;

-- name: ListFilesBySubjectID :many
SELECT
  id AS file_id,
  subject_id,
  user_id,
  original_filename AS name,
  gcs_object_path AS storage_path,
  mime_type,
  file_size_bytes AS size_bytes,
  status,
  NULL::text AS error_message,
  created_at AS uploaded_at,
  processed_at
FROM raw_files
WHERE subject_id = $1
  AND is_active = TRUE
ORDER BY created_at DESC;

-- name: CreateFile :one
INSERT INTO raw_files (
  id,
  subject_id,
  user_id,
  original_filename,
  file_type,
  file_size_bytes,
  gcs_bucket,
  gcs_object_path,
  mime_type,
  status
)
VALUES (
  sqlc.arg(file_id),
  sqlc.arg(subject_id),
  sqlc.arg(user_id),
  sqlc.arg(name),
  CASE
    WHEN lower(sqlc.arg(mime_type)) LIKE 'application/pdf%' THEN 'pdf'::file_type
    WHEN lower(sqlc.arg(mime_type)) LIKE 'text/%' THEN 'text'::file_type
    ELSE 'other'::file_type
  END,
  sqlc.arg(size_bytes),
  'materials',
  sqlc.arg(storage_path),
  sqlc.arg(mime_type),
  sqlc.arg(status)::file_status
)
RETURNING
  id AS file_id,
  subject_id,
  user_id,
  original_filename AS name,
  gcs_object_path AS storage_path,
  mime_type,
  file_size_bytes AS size_bytes,
  status,
  NULL::text AS error_message,
  created_at AS uploaded_at,
  processed_at;

-- name: UpdateFileStatus :one
UPDATE raw_files
SET
  status = sqlc.arg(status)::file_status,
  processed_at = CASE
    WHEN sqlc.arg(status)::file_status = 'completed' THEN NOW()
    ELSE processed_at
  END,
  updated_at = NOW()
WHERE id = sqlc.arg(file_id)
  AND is_active = TRUE
RETURNING
  id AS file_id,
  subject_id,
  user_id,
  original_filename AS name,
  gcs_object_path AS storage_path,
  mime_type,
  file_size_bytes AS size_bytes,
  status,
  NULL::text AS error_message,
  created_at AS uploaded_at,
  processed_at;

-- name: DeleteFile :exec
UPDATE raw_files
SET is_active = FALSE,
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(file_id)
  AND user_id = sqlc.arg(user_id)
  AND is_active = TRUE;
