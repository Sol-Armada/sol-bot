-- name: GetAttendanceByID :one
SELECT *
FROM attendance
WHERE id = $1;

-- name: ListAttendancePage :many
SELECT *
FROM attendance
ORDER BY date_created DESC
OFFSET sqlc.arg(offset_rows)::int
LIMIT sqlc.arg(limit_rows)::int;

-- name: ListAllAttendance :many
SELECT *
FROM attendance
ORDER BY date_created DESC;

-- name: ListActiveAttendance :many
SELECT *
FROM attendance
WHERE recorded = FALSE
   OR status = 'active'
   OR status = 'reverted'
ORDER BY date_created DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: ListRecordedAttendance :many
SELECT *
FROM attendance
WHERE recorded = FALSE
   OR status = 'recorded'
ORDER BY date_created DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: CountRecordedMemberAttendanceAfterJoin :one
SELECT COUNT(*)::int AS count
FROM attendance a
INNER JOIN attendance_participants ap ON ap.attendance_id = a.id
INNER JOIN members m ON m.id = ap.member_id
WHERE ap.member_id = sqlc.arg(member_id)
  AND a.recorded = TRUE
  AND a.date_created > m.joined;

-- name: CountUniqueAttendanceMembersSince :one
SELECT COUNT(DISTINCT ap.member_id)::int AS count
FROM attendance a
INNER JOIN attendance_participants ap ON ap.attendance_id = a.id
WHERE a.date_created >= sqlc.arg(since);

-- name: UpsertAttendance :exec
INSERT INTO attendance (
    id,
    name,
    submitted_by,
    recorded,
    successful,
    active,
    tokenable,
    status,
    channel_id,
    message_id,
    date_created,
    date_updated
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.narg(submitted_by),
    sqlc.arg(recorded),
    sqlc.arg(successful),
    sqlc.arg(active),
    sqlc.arg(tokenable),
    sqlc.arg(status),
    sqlc.arg(channel_id),
    sqlc.arg(message_id),
    sqlc.arg(date_created),
    sqlc.arg(date_updated)
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    submitted_by = EXCLUDED.submitted_by,
    recorded = EXCLUDED.recorded,
    successful = EXCLUDED.successful,
    active = EXCLUDED.active,
    tokenable = EXCLUDED.tokenable,
    status = EXCLUDED.status,
    channel_id = EXCLUDED.channel_id,
    message_id = EXCLUDED.message_id,
    date_created = EXCLUDED.date_created,
    date_updated = EXCLUDED.date_updated;

-- name: UpsertAttendanceParticipant :exec
INSERT INTO attendance_participants (
    attendance_id,
    member_id,
    joined_at_start,
    stayed_until_end,
    has_issue,
    updated_at,
    is_manager
	)
VALUES (
    sqlc.arg(attendance_id),
    sqlc.arg(member_id),
    sqlc.arg(joined_at_start),
    sqlc.arg(stayed_until_end),
    sqlc.arg(has_issue),
    COALESCE(sqlc.arg(updated_at), NOW()),
    sqlc.arg(is_manager)
	)
ON CONFLICT (attendance_id, member_id) DO UPDATE
SET joined_at_start = EXCLUDED.joined_at_start,
    stayed_until_end = EXCLUDED.stayed_until_end,
    has_issue = EXCLUDED.has_issue,
    updated_at = EXCLUDED.updated_at,
    is_manager = EXCLUDED.is_manager;

-- name: DeleteAttendanceParticipant :exec
DELETE FROM attendance_participants
WHERE attendance_id = sqlc.arg(attendance_id)
  AND member_id = sqlc.arg(member_id);

-- name: ListAttendanceParticipants :many
SELECT *
FROM attendance_participants
WHERE attendance_id = $1
ORDER BY member_id;

-- name: DeleteAttendance :exec
DELETE FROM attendance
WHERE id = $1;
