-- name: InsertToken :exec
INSERT INTO tokens (
    id,
    member_id,
    amount,
    reason,
    attendance_id,
    comment,
    giver_id,
  created_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(member_id),
    sqlc.arg(amount),
    sqlc.arg(reason),
    sqlc.narg(attendance_id),
    sqlc.narg(comment),
    sqlc.narg(giver_id),
  sqlc.arg(created_at)
);

-- name: GetTokenByID :one
SELECT *
FROM tokens
WHERE id = $1;

-- name: ListAllTokens :many
SELECT *
FROM tokens
ORDER BY created_at DESC;

-- name: ListTokensByAttendanceID :many
SELECT *
FROM tokens
WHERE attendance_id = $1
ORDER BY created_at DESC;

-- name: ListTokensByMemberAndAttendance :many
SELECT *
FROM tokens
WHERE member_id = sqlc.arg(member_id)
  AND attendance_id = sqlc.arg(attendance_id)
ORDER BY created_at DESC;

-- name: GetTokenBalances :many
SELECT member_id,
       SUM(amount)::int AS balance
FROM tokens
GROUP BY member_id
ORDER BY member_id;
