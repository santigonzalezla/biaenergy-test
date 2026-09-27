-- name: GetUserByEmail :one
SELECT *
FROM app_user
WHERE str_email_user = sqlc.arg('email');

-- name: GetUser :one
SELECT *
FROM app_user
WHERE uid_user = sqlc.arg('id');

-- name: UpsertUser :one
INSERT INTO app_user (str_email_user, str_password_user, str_name_user)
VALUES (sqlc.arg('email'), sqlc.arg('password_hash'), sqlc.arg('name'))
ON CONFLICT (str_email_user) DO UPDATE
    SET str_password_user = EXCLUDED.str_password_user,
        str_name_user     = EXCLUDED.str_name_user
RETURNING *;
