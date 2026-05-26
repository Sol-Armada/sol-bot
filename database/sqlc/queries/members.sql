-- name: GetMember :one
SELECT *
FROM members
WHERE id = $1;

-- name: ListMembersByIDs :many
SELECT *
FROM members
WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: ListMembersPage :many
SELECT *
FROM members
WHERE (NOT sqlc.arg(exclude_bots)::boolean OR is_bot = FALSE)
ORDER BY id
OFFSET sqlc.arg(offset_rows)::int
LIMIT sqlc.arg(limit_rows)::int;

-- name: ListMembersByBlueprint :many
SELECT m.*
FROM members m
INNER JOIN member_blueprints mb ON mb.member_id = m.id
WHERE mb.blueprint_id = sqlc.arg(blueprint_id)
ORDER BY m.id;

-- name: ListRandomMembersByRank :many
SELECT *
FROM members
WHERE rank <= sqlc.arg(max_rank)::int
  AND rank <> 0
ORDER BY random()
LIMIT sqlc.arg(limit_rows)::int;

-- name: UpsertMember :exec
INSERT INTO members (
    id,
    name,
    rank,
    joined,
    updated,
    is_bot,
    is_ally,
  is_affiliate,
  is_guest
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.arg(rank),
    sqlc.arg(joined),
    sqlc.arg(updated),
    sqlc.arg(is_bot),
    sqlc.arg(is_ally),
    sqlc.arg(is_affiliate),
  sqlc.arg(is_guest)
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    rank = EXCLUDED.rank,
    joined = EXCLUDED.joined,
    updated = EXCLUDED.updated,
    is_bot = EXCLUDED.is_bot,
    is_ally = EXCLUDED.is_ally,
    is_affiliate = EXCLUDED.is_affiliate,
  is_guest = EXCLUDED.is_guest;

-- name: ReplaceMemberBlueprints :exec
DELETE FROM member_blueprints
WHERE member_id = sqlc.arg(member_id);

-- name: AddMemberBlueprint :exec
INSERT INTO member_blueprints (member_id, blueprint_id)
VALUES (sqlc.arg(member_id), sqlc.arg(blueprint_id))
ON CONFLICT (member_id, blueprint_id) DO NOTHING;

-- name: DeleteMember :exec
DELETE FROM members
WHERE id = $1;

-- name: GetMemberIDs :many
SELECT id
FROM members
ORDER BY id;
