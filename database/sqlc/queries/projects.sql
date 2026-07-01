-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = sqlc.arg(id);

-- name: UpsertProject :one
INSERT INTO projects (id, name, description, owner_id, status_id, due_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(owner_id), sqlc.arg(status_id), sqlc.arg(due_at), sqlc.arg(created_at), sqlc.arg(updated_at))
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    status_id = EXCLUDED.status_id,
    due_at = EXCLUDED.due_at,
    updated_at = EXCLUDED.updated_at
RETURNING *;

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

-- name: ListProjectTasks :many
SELECT * FROM project_kanban_tasks WHERE project_id = sqlc.arg(project_id) ORDER BY position ASC;

-- name: GetProjectTaskByID :one
SELECT * FROM project_kanban_tasks WHERE id = sqlc.arg(id);

-- name: ListProjectTasksByAssignee :many
SELECT * FROM project_kanban_tasks WHERE assignee_id = sqlc.arg(assignee_id) ORDER BY position ASC;

-- name: UpsertProjectTask :one
INSERT INTO project_kanban_tasks (id, project_id, status_name, title, description, position, priority, due_at, assignee_id, parent_task_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(project_id), sqlc.arg(status_name), sqlc.arg(title), sqlc.arg(description), sqlc.arg(position), sqlc.arg(priority), sqlc.arg(due_at), sqlc.arg(assignee_id), sqlc.arg(parent_task_id), sqlc.arg(created_at), sqlc.arg(updated_at))
ON CONFLICT (id) DO UPDATE SET
    status_name = EXCLUDED.status_name,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    position = EXCLUDED.position,
    priority = EXCLUDED.priority,
    due_at = EXCLUDED.due_at,
    assignee_id = EXCLUDED.assignee_id,
    parent_task_id = EXCLUDED.parent_task_id,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteProjectTask :exec
DELETE FROM project_kanban_tasks WHERE id = sqlc.arg(id);

-- name: ListProjectTaskHistory :many
SELECT * FROM project_kanban_task_history WHERE task_id = sqlc.arg(task_id) ORDER BY performed_at DESC;

-- name: InsertProjectTaskHistory :one
INSERT INTO project_kanban_task_history (task_id, action, performed_by_id, performed_at, details)
VALUES (sqlc.arg(task_id), sqlc.arg(action), sqlc.arg(performed_by_id), sqlc.arg(performed_at), sqlc.arg(details))
RETURNING *;
