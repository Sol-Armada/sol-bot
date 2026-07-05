-- name: InsertRaffle :one
INSERT INTO raffles (
    name,
    attendance_id,
    prize,
    quantity,
    tickets_json,
    winners,
    ended,
    test,
    channel_id,
    message_id,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(name),
    sqlc.arg(attendance_id),
    sqlc.arg(prize),
    sqlc.arg(quantity),
    sqlc.arg(tickets_json),
    COALESCE(sqlc.arg(winners)::TEXT[], '{}'::TEXT[]),
    sqlc.arg(ended),
    sqlc.arg(test),
    sqlc.arg(channel_id),
    sqlc.arg(message_id),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
RETURNING id;

-- name: UpdateRaffle :exec
UPDATE raffles
SET
    name = sqlc.arg(name),
    attendance_id = sqlc.arg(attendance_id),
    prize = sqlc.arg(prize),
    quantity = sqlc.arg(quantity),
    tickets_json = sqlc.arg(tickets_json),
    winners = COALESCE(sqlc.arg(winners)::TEXT[], '{}'::TEXT[]),
    ended = sqlc.arg(ended),
    test = sqlc.arg(test),
    channel_id = sqlc.arg(channel_id),
    message_id = sqlc.arg(message_id),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: GetRaffleById :one
SELECT *
FROM raffles
WHERE id = sqlc.arg(id);

-- name: ListRaffles :many
SELECT *
FROM raffles
ORDER BY created_at DESC;

-- name: DeleteRaffleById :exec
DELETE FROM raffles
WHERE id = sqlc.arg(id);

-- name: GetLatestRaffle :one
SELECT *
FROM raffles
ORDER BY created_at DESC
LIMIT 1;
