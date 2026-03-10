-- sql/queries/subjects.sql

-- name: ListSubjectsByUserID :many
SELECT
  id AS subject_id,
  owner_user_id AS user_id,
  title AS name,
  course_code AS lms_course_id,
  created_at,
  updated_at
FROM subjects
WHERE owner_user_id = $1
  AND is_active = TRUE
ORDER BY created_at DESC;

-- name: GetSubjectByID :one
SELECT
  id AS subject_id,
  owner_user_id AS user_id,
  title AS name,
  course_code AS lms_course_id,
  created_at,
  updated_at
FROM subjects
WHERE id = $1
  AND is_active = TRUE;

-- name: GetSubjectByIDAndUserID :one
SELECT
  id AS subject_id,
  owner_user_id AS user_id,
  title AS name,
  course_code AS lms_course_id,
  created_at,
  updated_at
FROM subjects
WHERE id = $1
  AND owner_user_id = $2
  AND is_active = TRUE;

-- name: CreateSubject :one
INSERT INTO subjects (id, owner_user_id, title, course_code)
VALUES (sqlc.arg(subject_id), sqlc.arg(user_id), sqlc.arg(name), sqlc.narg(lms_course_id))
RETURNING
  id AS subject_id,
  owner_user_id AS user_id,
  title AS name,
  course_code AS lms_course_id,
  created_at,
  updated_at;

-- name: UpdateSubjectName :one
UPDATE subjects
SET title = sqlc.arg(name),
    updated_at = NOW()
WHERE id = sqlc.arg(subject_id)
  AND owner_user_id = sqlc.arg(user_id)
  AND is_active = TRUE
RETURNING
  id AS subject_id,
  owner_user_id AS user_id,
  title AS name,
  course_code AS lms_course_id,
  created_at,
  updated_at;

-- name: DeleteSubject :exec
UPDATE subjects
SET is_active = FALSE,
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(subject_id)
  AND owner_user_id = sqlc.arg(user_id)
  AND is_active = TRUE;
