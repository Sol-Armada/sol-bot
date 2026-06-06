-- name: GetRollEventByID :one
SELECT *
FROM roll_events
WHERE id = $1;

-- name: ListActiveRollEvents :many
SELECT *
FROM roll_events
WHERE ended = FALSE
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: UpsertRollEvent :exec
INSERT INTO roll_events (
    id,
    name,
    attendance_id,
    end_time,
    ended,
    channel_id,
    embed_message_id,
    input_message_id,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.narg(attendance_id),
    sqlc.narg(end_time),
    sqlc.arg(ended),
    sqlc.arg(channel_id),
    sqlc.arg(embed_message_id),
    sqlc.arg(input_message_id),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    attendance_id = EXCLUDED.attendance_id,
    end_time = EXCLUDED.end_time,
    ended = EXCLUDED.ended,
    channel_id = EXCLUDED.channel_id,
    embed_message_id = EXCLUDED.embed_message_id,
    input_message_id = EXCLUDED.input_message_id,
    updated_at = EXCLUDED.updated_at;

-- name: MarkRollEventEnded :exec
UPDATE roll_events
SET ended = TRUE,
    updated_at = COALESCE(sqlc.arg(updated_at), NOW())
WHERE id = sqlc.arg(id);

-- name: DeleteRollEvent :exec
DELETE FROM roll_events
WHERE id = $1;

-- name: UpsertRollItem :exec
INSERT INTO roll_items (
    id,
    roll_event_id,
    name,
    amount,
    sort_order,
    channel_id,
    message_id,
    created_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(roll_event_id),
    sqlc.arg(name),
    sqlc.arg(amount),
    sqlc.arg(sort_order),
    sqlc.arg(channel_id),
    sqlc.arg(message_id),
    sqlc.arg(created_at)
)
ON CONFLICT (id) DO UPDATE
SET roll_event_id = EXCLUDED.roll_event_id,
    name = EXCLUDED.name,
    amount = EXCLUDED.amount,
    sort_order = EXCLUDED.sort_order;

-- name: ListRollItemsByEvent :many
SELECT *
FROM roll_items
WHERE roll_event_id = sqlc.arg(roll_event_id)
ORDER BY sort_order ASC, created_at ASC;

-- name: DeleteRollItemsByEvent :exec
DELETE FROM roll_items
WHERE roll_event_id = sqlc.arg(roll_event_id);

-- name: UpsertRollEntry :exec
INSERT INTO roll_entries (
    roll_event_id,
    roll_item_id,
    member_id,
    choice,
    roll_value,
    winner,
    created_at,
    updated_at
)
VALUES (
    sqlc.arg(roll_event_id),
    sqlc.arg(roll_item_id),
    sqlc.arg(member_id),
    sqlc.arg(choice),
    sqlc.narg(roll_value),
    sqlc.arg(winner),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (roll_item_id, member_id) DO UPDATE
SET choice = EXCLUDED.choice,
    roll_value = EXCLUDED.roll_value,
    winner = EXCLUDED.winner,
    updated_at = EXCLUDED.updated_at;

-- name: ListRollEntriesByEvent :many
SELECT *
FROM roll_entries
WHERE roll_event_id = sqlc.arg(roll_event_id)
ORDER BY roll_item_id ASC, created_at ASC;

-- name: ListRollEntriesByItem :many
SELECT *
FROM roll_entries
WHERE roll_item_id = sqlc.arg(roll_item_id)
ORDER BY created_at ASC;

-- name: DeleteRollEntry :exec
DELETE FROM roll_entries
WHERE roll_item_id = sqlc.arg(roll_item_id)
  AND member_id = sqlc.arg(member_id);

-- name: DeleteRollEntriesByEvent :exec
DELETE FROM roll_entries
WHERE roll_event_id = sqlc.arg(roll_event_id);
