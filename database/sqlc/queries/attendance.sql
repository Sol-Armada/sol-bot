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
INNER JOIN attendance_members am ON am.attendance_id = a.id
INNER JOIN members m ON m.id = am.member_id
WHERE am.member_id = sqlc.arg(member_id)
  AND a.recorded = TRUE
  AND a.date_created > m.joined;

-- name: CountUniqueAttendanceMembersSince :one
SELECT COUNT(DISTINCT am.member_id)::int AS count
FROM attendance a
INNER JOIN attendance_members am ON am.attendance_id = a.id
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
    payouts_total,
    payouts_per_member,
    payouts_org_take,
    from_start,
    stayed,
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
    sqlc.narg(payouts_total),
    sqlc.narg(payouts_per_member),
    sqlc.narg(payouts_org_take),
    sqlc.arg(from_start),
    sqlc.arg(stayed),
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
    payouts_total = EXCLUDED.payouts_total,
    payouts_per_member = EXCLUDED.payouts_per_member,
    payouts_org_take = EXCLUDED.payouts_org_take,
    from_start = EXCLUDED.from_start,
    stayed = EXCLUDED.stayed,
    channel_id = EXCLUDED.channel_id,
    message_id = EXCLUDED.message_id,
    date_created = EXCLUDED.date_created,
    date_updated = EXCLUDED.date_updated;

-- name: ReplaceAttendanceMembers :exec
DELETE FROM attendance_members
WHERE attendance_id = sqlc.arg(attendance_id);

-- name: ReplaceAttendanceIssues :exec
DELETE FROM attendance_issues
WHERE attendance_id = sqlc.arg(attendance_id);

-- name: AddAttendanceMember :exec
INSERT INTO attendance_members (attendance_id, member_id)
VALUES (sqlc.arg(attendance_id), sqlc.arg(member_id))
ON CONFLICT (attendance_id, member_id) DO NOTHING;

-- name: AddAttendanceIssue :exec
INSERT INTO attendance_issues (attendance_id, member_id)
VALUES (sqlc.arg(attendance_id), sqlc.arg(member_id))
ON CONFLICT (attendance_id, member_id) DO NOTHING;

-- name: ListAttendanceMemberIDs :many
SELECT member_id
FROM attendance_members
WHERE attendance_id = $1
ORDER BY member_id;

-- name: ListAttendanceIssueIDs :many
SELECT member_id
FROM attendance_issues
WHERE attendance_id = $1
ORDER BY member_id;

-- name: DeleteAttendance :exec
DELETE FROM attendance
WHERE id = $1;
