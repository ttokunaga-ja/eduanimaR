-- sql/queries/ingest_jobs.sql

-- name: CreateIngestJob :one
INSERT INTO ingest_jobs (job_id, file_id, status, max_retries)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetIngestJobByFileID :one
-- 最新の ingest_job を取得（ファイルのステータス確認用）
SELECT *
FROM ingest_jobs
WHERE file_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetIngestJobByID :one
SELECT *
FROM ingest_jobs
WHERE job_id = $1;

-- name: UpdateIngestJobStatus :one
UPDATE ingest_jobs
SET
    status        = $2,
    error_message = $3,
    started_at    = CASE
                        WHEN $2::job_status = 'processing' THEN NOW()
                        ELSE started_at
                    END,
    completed_at  = CASE
                        WHEN $2::job_status IN ('completed', 'failed') THEN NOW()
                        ELSE completed_at
                    END,
    retry_count   = CASE
                        WHEN $2::job_status = 'failed' THEN retry_count + 1
                        ELSE retry_count
                    END
WHERE job_id = $1
RETURNING *;
