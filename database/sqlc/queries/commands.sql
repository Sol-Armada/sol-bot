-- name: InsertCommandLog :exec
INSERT INTO command_logs (
    name,
    occurred_at,
    user_id,
    interaction_type,
    button_id,
    error_text,
    options_json
)
VALUES (
    sqlc.arg(name),
    sqlc.arg(occurred_at),
    sqlc.arg(user_id),
    sqlc.arg(interaction_type),
    sqlc.arg(button_id),
    sqlc.arg(error_text),
    sqlc.arg(options_json)
);

-- name: GetWeeklyCommandCounts :many
SELECT name,
       COUNT(*)::bigint AS count
FROM command_logs
WHERE occurred_at >= NOW() - INTERVAL '7 days'
GROUP BY name
ORDER BY count DESC, name ASC;
