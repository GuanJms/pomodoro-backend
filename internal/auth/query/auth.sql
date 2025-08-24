-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    password_hash,
    role
) VALUES (
    sqlc.arg(username),
    sqlc.arg(email),
    sqlc.arg(password_hash),
    sqlc.arg(role)
)
RETURNING id, username, email, role, created_at;

-- name: GetUserByUsername :one
SELECT id, username, email, role, created_at FROM users WHERE username = sqlc.arg(username);

-- name: GetUserByEmail :one
SELECT id, username, email, role, created_at FROM users WHERE email = sqlc.arg(email);

-- name: GetPasswordHashByUsername :one
SELECT password_hash FROM users WHERE username = sqlc.arg(username);
