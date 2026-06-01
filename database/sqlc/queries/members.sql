-- name: GetMember :one
SELECT 
  m.*,
  ri.primary_org,
  ri.affiliations
FROM members m
LEFT JOIN rsi_info ri ON ri.handle = m.name
WHERE id = $1;

-- name: ListMembersByIDs :many
SELECT 
  m.*,
  ri.primary_org,
  ri.affiliations
FROM members m
LEFT JOIN rsi_info ri ON ri.handle = m.name
WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: ListMembersPage :many
SELECT 
  m.*,
  ri.primary_org,
  ri.affiliations
FROM members m
LEFT JOIN rsi_info ri ON ri.handle = m.name
WHERE (NOT sqlc.arg(exclude_bots)::boolean OR is_bot = FALSE)
ORDER BY id
OFFSET sqlc.arg(offset_rows)::int
LIMIT sqlc.arg(limit_rows)::int;

-- name: ListMembersByBlueprint :many
SELECT 
  m.*,
  ri.primary_org,
  ri.affiliations
FROM members m
LEFT JOIN rsi_info ri ON ri.handle = m.name
INNER JOIN member_blueprints mb ON mb.member_id = m.id
WHERE mb.blueprint_id = sqlc.arg(blueprint_id)
ORDER BY m.id;

-- name: ListRandomMembersByRank :many
SELECT 
  m.*,
  ri.primary_org,
  ri.affiliations
FROM members m
LEFT JOIN rsi_info ri ON ri.handle = m.name
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
  dm_opt_out
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.arg(rank),
    sqlc.arg(joined),
    COALESCE(sqlc.arg(updated), NOW()),
    sqlc.arg(is_bot),
    sqlc.arg(is_ally),
    sqlc.arg(is_affiliate),
  sqlc.arg(dm_opt_out)
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    rank = EXCLUDED.rank,
    joined = EXCLUDED.joined,
    updated = EXCLUDED.updated,
    is_bot = EXCLUDED.is_bot,
    is_ally = EXCLUDED.is_ally,
    is_affiliate = EXCLUDED.is_affiliate,
  dm_opt_out = EXCLUDED.dm_opt_out;

-- name: ReplaceMemberBlueprints :exec
DELETE FROM member_blueprints
WHERE member_id = sqlc.arg(member_id);

-- name: AddMemberBlueprint :exec
INSERT INTO member_blueprints (member_id, blueprint_id)
VALUES (sqlc.arg(member_id), sqlc.arg(blueprint_id))
ON CONFLICT (member_id, blueprint_id) DO NOTHING;

-- name: DeleteMember :exec
UPDATE members
SET date_left = COALESCE(sqlc.arg(date_left), NOW()),
    reason_left = sqlc.arg(reason_left)
WHERE id = sqlc.arg(id);

-- name: GetMemberIDs :many
SELECT id
FROM members
ORDER BY id;

-- name: ListPromotions :many
WITH attendance_counts AS (
  SELECT
    ap.member_id,
    COUNT(*)::int AS attendance_count
  FROM attendance_participants ap
  JOIN attendance a ON a.id = ap.attendance_id
  JOIN members m2 ON m2.id = ap.member_id
  WHERE a.recorded = TRUE
    AND a.date_created > m2.joined
  GROUP BY ap.member_id
)
SELECT
  m.id,
  m.name,
  m.rank AS current_rank,
  COALESCE(ac.attendance_count, 0) AS attendance_count,
  CASE
    WHEN m.rank = 7 AND COALESCE(ac.attendance_count, 0) >= 3 THEN 6
    WHEN m.rank = 6 AND COALESCE(ac.attendance_count, 0) >= 10 THEN 5
    WHEN m.rank = 5 AND COALESCE(ac.attendance_count, 0) >= 20 THEN 4
    ELSE NULL
  END::int AS next_rank
FROM members m
LEFT JOIN attendance_counts ac ON ac.member_id = m.id
LEFT JOIN rsi_info ri ON ri.handle = m.name
WHERE m.rank <= 7
  AND m.rank > 0
  AND ri.primary_org_sid = 'SOLARMADA'
  AND (
    (m.rank = 7 AND COALESCE(ac.attendance_count, 0) >= 3) OR
    (m.rank = 6 AND COALESCE(ac.attendance_count, 0) >= 10) OR
    (m.rank = 5 AND COALESCE(ac.attendance_count, 0) >= 20)
  )
ORDER BY attendance_count DESC, m.id;
