-- name: ListAttendanceTags :many
SELECT tag
FROM attendance_tags
ORDER BY tag;

-- name: InsertAttendanceTag :exec
INSERT INTO attendance_tags (tag)
VALUES (sqlc.arg(tag))
ON CONFLICT (tag) DO NOTHING;

-- name: DeleteAttendanceTag :exec
DELETE FROM attendance_tags
WHERE tag = sqlc.arg(tag);

-- name: DeleteAllAttendanceTags :exec
DELETE FROM attendance_tags;

-- name: ListAttendanceNames :many
SELECT name
FROM attendance_names
ORDER BY name;

-- name: InsertAttendanceName :exec
INSERT INTO attendance_names (name)
VALUES (sqlc.arg(name))
ON CONFLICT (name) DO NOTHING;

-- name: DeleteAttendanceName :exec
DELETE FROM attendance_names
WHERE name = sqlc.arg(name);

-- name: DeleteAllAttendanceNames :exec
DELETE FROM attendance_names;
