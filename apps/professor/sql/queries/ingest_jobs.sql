-- sql/queries/ingest_jobs.sql

-- name: CreateIngestJob :one
INSERT INTO jobs (id, target_raw_file_id, status, max_retries, idempotency_key, job_type)
VALUES ($1, $2, $3, $4, $5, 'file_ingestion')
RETURNING
  id AS job_id,
  target_raw_file_id AS file_id,
  status,
  retry_count,
  max_retries,
  error_message,
  created_at,
  started_at,
  completed_at;

-- name: GetIngestJobByFileID :one
SELECT
  id AS job_id,
  target_raw_file_id AS file_id,
  status,
  retry_count,
  max_retries,
  error_message,
  created_at,
  started_at,
  completed_at
FROM jobs
WHERE target_raw_file_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetIngestJobByID :one
SELECT
  id AS job_id,
  target_raw_file_id AS file_id,
  status,
  retry_count,
  max_retries,
  error_message,
  created_at,
  started_at,
  completed_at
FROM jobs
WHERE id = $1;

-- name: UpdateIngestJobStatus :one
UPDATE jobs
SET
  status = $2,
  error_message = $3,
  started_at = CASE WHEN $2::job_status = 'processing' THEN NOW() ELSE started_at END,
  completed_at = CASE WHEN $2::job_status IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
  retry_count = CASE WHEN $2::job_status = 'failed' THEN retry_count + 1 ELSE retry_count END,
  updated_at = NOW()
WHERE id = $1
RETURNING
  id AS job_id,
  target_raw_file_id AS file_id,
  status,
  retry_count,
  max_retries,
  error_message,
  created_at,
  started_at,
  completed_at;
