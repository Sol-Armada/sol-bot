-- name: UpsertSOSTicket :exec
INSERT INTO sos_tickets (id, member_id, payload_json, updated_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(member_id),
    sqlc.arg(payload_json),
    sqlc.arg(updated_at)
)
ON CONFLICT (id) DO UPDATE
SET member_id = EXCLUDED.member_id,
    payload_json = EXCLUDED.payload_json,
    updated_at = EXCLUDED.updated_at;

-- name: UpsertKanbanCard :exec
INSERT INTO kanban_cards (id, payload_json, updated_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(payload_json),
    sqlc.arg(updated_at)
)
ON CONFLICT (id) DO UPDATE
SET payload_json = EXCLUDED.payload_json,
    updated_at = EXCLUDED.updated_at;

-- name: UpsertBlueprintDoc :exec
INSERT INTO blueprint_docs (id, payload_json, updated_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(payload_json),
    sqlc.arg(updated_at)
)
ON CONFLICT (id) DO UPDATE
SET payload_json = EXCLUDED.payload_json,
    updated_at = EXCLUDED.updated_at;
