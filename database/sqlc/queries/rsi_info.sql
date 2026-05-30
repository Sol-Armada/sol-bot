-- name: UpsertRsiInfo :exec
INSERT INTO rsi_info (handle, primary_org, primary_org_sid, affiliations)
VALUES (sqlc.arg(handle), sqlc.arg(primary_org), sqlc.arg(primary_org_sid), sqlc.arg(affiliations))
ON CONFLICT (handle) DO UPDATE
SET primary_org = EXCLUDED.primary_org,
    primary_org_sid = EXCLUDED.primary_org_sid,
    affiliations = EXCLUDED.affiliations;

-- name: GetRsiInfoByHandle :one
SELECT * FROM rsi_info WHERE handle = sqlc.arg(handle);

-- name: ListRsiInfoByHandles :many
SELECT * FROM rsi_info WHERE handle = ANY(sqlc.arg(handles)::text[]);
