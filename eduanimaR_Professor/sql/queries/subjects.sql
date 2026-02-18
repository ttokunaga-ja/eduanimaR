-- sql/queries/subjects.sql

-- name: ListSubjectsByUserID :many
SELECT *
FROM subjects
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetSubjectByID :one
SELECT *
FROM subjects
WHERE subject_id = $1;

-- name: GetSubjectByIDAndUserID :one
SELECT *
FROM subjects
WHERE subject_id = $1
  AND user_id    = $2;

-- name: CreateSubject :one
INSERT INTO subjects (subject_id, user_id, name, lms_course_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSubjectName :one
UPDATE subjects
SET name       = $2,
    updated_at = NOW()
WHERE subject_id = $1
  AND user_id    = $3
RETURNING *;

-- name: DeleteSubject :exec
DELETE FROM subjects
WHERE subject_id = $1
  AND user_id    = $2;
