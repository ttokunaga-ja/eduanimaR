-- sql/queries/qa_sessions.sql

-- name: CreateQASession :one
INSERT INTO qa_sessions (session_id, user_id, subject_id, question)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetQASessionByID :one
SELECT *
FROM qa_sessions
WHERE session_id = $1;

-- name: GetQASessionByIDAndUserID :one
SELECT *
FROM qa_sessions
WHERE session_id = $1
  AND user_id    = $2;

-- name: ListQASessionsBySubjectID :many
SELECT
    session_id, user_id, subject_id,
    question, answer, sources, feedback,
    created_at, answered_at
FROM qa_sessions
WHERE subject_id = $1
  AND user_id    = $2
ORDER BY created_at DESC
LIMIT  $3
OFFSET $4;

-- name: CountQASessionsBySubjectID :one
SELECT COUNT(*)
FROM qa_sessions
WHERE subject_id = $1
  AND user_id    = $2;

-- name: UpdateQASessionAnswer :one
UPDATE qa_sessions
SET
    answer      = $2,
    sources     = $3,
    answered_at = NOW()
WHERE session_id = $1
RETURNING *;

-- name: UpdateQASessionFeedback :one
UPDATE qa_sessions
SET feedback = $2
WHERE session_id = $1
  AND user_id    = $3
RETURNING *;
