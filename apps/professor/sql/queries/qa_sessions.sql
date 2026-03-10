-- sql/queries/qa_sessions.sql

-- name: CreateQASession :one
INSERT INTO chats (id, user_id, subject_id, question)
VALUES ($1, $2, $3, $4)
RETURNING
  id AS session_id,
  user_id,
  subject_id,
  question,
  final_answer_markdown AS answer,
  evidence_snippets AS sources,
  COALESCE(CASE feedback WHEN 'good' THEN 1 WHEN 'bad' THEN -1 END, 0)::smallint AS feedback,
  created_at,
  completed_at AS answered_at;

-- name: GetQASessionByID :one
SELECT
  id AS session_id,
  user_id,
  subject_id,
  question,
  final_answer_markdown AS answer,
  evidence_snippets AS sources,
  COALESCE(CASE feedback WHEN 'good' THEN 1 WHEN 'bad' THEN -1 END, 0)::smallint AS feedback,
  created_at,
  completed_at AS answered_at
FROM chats
WHERE id = $1
  AND is_active = TRUE;

-- name: GetQASessionByIDAndUserID :one
SELECT
  id AS session_id,
  user_id,
  subject_id,
  question,
  final_answer_markdown AS answer,
  evidence_snippets AS sources,
  COALESCE(CASE feedback WHEN 'good' THEN 1 WHEN 'bad' THEN -1 END, 0)::smallint AS feedback,
  created_at,
  completed_at AS answered_at
FROM chats
WHERE id = $1
  AND user_id = $2
  AND is_active = TRUE;

-- name: ListQASessionsBySubjectID :many
SELECT
  id AS session_id,
  user_id,
  subject_id,
  question,
  final_answer_markdown AS answer,
  evidence_snippets AS sources,
  COALESCE(CASE feedback WHEN 'good' THEN 1 WHEN 'bad' THEN -1 END, 0)::smallint AS feedback,
  created_at,
  completed_at AS answered_at
FROM chats
WHERE subject_id = $1
  AND user_id = $2
  AND is_active = TRUE
ORDER BY created_at DESC
LIMIT $3
OFFSET $4;

-- name: CountQASessionsBySubjectID :one
SELECT COUNT(*)
FROM chats
WHERE subject_id = $1
  AND user_id = $2
  AND is_active = TRUE;

-- name: UpdateQASessionAnswer :one
UPDATE chats
SET
  final_answer_markdown = $2,
  evidence_snippets = $3,
  completed_at = NOW(),
  updated_at = NOW()
WHERE id = $1
  AND is_active = TRUE
RETURNING
  id AS session_id,
  user_id,
  subject_id,
  question,
  final_answer_markdown AS answer,
  evidence_snippets AS sources,
  COALESCE(CASE feedback WHEN 'good' THEN 1 WHEN 'bad' THEN -1 END, 0)::smallint AS feedback,
  created_at,
  completed_at AS answered_at;

-- name: UpdateQASessionFeedback :one
UPDATE chats
SET
  feedback = CASE
    WHEN $2 = 1 THEN 'good'::chat_feedback
    WHEN $2 = -1 THEN 'bad'::chat_feedback
    ELSE feedback
  END,
  feedback_at = NOW(),
  updated_at = NOW()
WHERE id = $1
  AND user_id = $3
  AND is_active = TRUE
RETURNING
  id AS session_id,
  user_id,
  subject_id,
  question,
  final_answer_markdown AS answer,
  evidence_snippets AS sources,
  COALESCE(CASE feedback WHEN 'good' THEN 1 WHEN 'bad' THEN -1 END, 0)::smallint AS feedback,
  created_at,
  completed_at AS answered_at;
