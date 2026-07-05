-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC;

-- name: GetProjectById :one
SELECT * FROM projects WHERE id = sqlc.arg(id);

-- name: InsertProject :one
INSERT INTO projects (
    id, 
    name, 
    description, 
    owner_id, 
    status_id, 
    due_at, 
    created_at, 
    updated_at
)
VALUES (
    sqlc.arg(id), 
    sqlc.arg(name), 
    sqlc.arg(description), 
    sqlc.arg(owner_id), 
    sqlc.arg(status_id), 
    sqlc.arg(due_at), 
    sqlc.arg(created_at), 
    sqlc.arg(updated_at)
)
RETURNING id;

-- name: UpdateProject :exec
UPDATE projects SET name = sqlc.arg(name), description = sqlc.arg(description), owner_id = sqlc.arg(owner_id), status_id = sqlc.arg(status_id), due_at = sqlc.arg(due_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = sqlc.arg(id);

-- name: ListProjectStatuses :many
SELECT * FROM project_statuses ORDER BY created_at DESC;

-- name: UpsertProjectStatus :one
INSERT INTO project_statuses (id, name, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(created_at), sqlc.arg(updated_at))
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: ListTaskStatuses :many
SELECT * FROM task_statuses WHERE project_id = sqlc.arg(project_id);

-- name: GetTaskStatus :one
SELECT * FROM task_statuses WHERE project_id = sqlc.arg(project_id) AND name = sqlc.arg(name);

-- name: InsertTaskStatus :one
INSERT INTO task_statuses (project_id, name, position, color, is_done_state)
VALUES (sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(position), sqlc.arg(color), sqlc.arg(is_done_state))
RETURNING *;

-- name: UpdateTaskStatus :exec
UPDATE task_statuses SET position = sqlc.arg(position), color = sqlc.arg(color), is_done_state = sqlc.arg(is_done_state)
WHERE project_id = sqlc.arg(project_id) AND name = sqlc.arg(name);

-- name: DeleteTaskStatus :exec
DELETE FROM task_statuses WHERE project_id = sqlc.arg(project_id) AND name = sqlc.arg(name);

-- name: ListTasks :many
SELECT * FROM tasks WHERE project_id = sqlc.arg(project_id) ORDER BY position ASC;

-- name: ListTasksByProject :many
SELECT * FROM tasks WHERE project_id = sqlc.arg(project_id) ORDER BY position ASC;

-- name: GetTaskById :one
SELECT * FROM tasks WHERE id = sqlc.arg(id);

-- name: ListTasksByAssignee :many
SELECT * FROM tasks WHERE assignee_id = sqlc.arg(assignee_id) ORDER BY position ASC;

-- name: InsertTask :one
INSERT INTO tasks (id, project_id, status_name, title, description, position, priority, due_at, assignee_id, parent_task_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(project_id), sqlc.arg(status_name), sqlc.arg(title), sqlc.arg(description), sqlc.arg(position), sqlc.arg(priority),
    sqlc.arg(due_at), sqlc.arg(assignee_id), sqlc.arg(parent_task_id), sqlc.arg(created_at), sqlc.arg(updated_at))
RETURNING id;

-- name: UpdateTask :exec
UPDATE tasks SET status_name = sqlc.arg(status_name), title = sqlc.arg(title), description = sqlc.arg(description), position = sqlc.arg(position), priority = sqlc.arg(priority), due_at = sqlc.arg(due_at), assignee_id = sqlc.arg(assignee_id), parent_task_id = sqlc.arg(parent_task_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: UpdateTaskPosition :exec
UPDATE tasks SET position = sqlc.arg(position), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = sqlc.arg(id);

-- name: ListTaskHistory :many
SELECT * FROM task_history WHERE task_id = sqlc.arg(task_id) ORDER BY performed_at DESC;

-- name: InsertTaskHistory :exec
INSERT INTO task_history (task_id, action, performed_by_id, performed_at, details)
VALUES (sqlc.arg(task_id), sqlc.arg(action), sqlc.arg(performed_by_id), sqlc.arg(performed_at), sqlc.arg(details)::jsonb);
