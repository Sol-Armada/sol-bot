CREATE TABLE IF NOT EXISTS project_statuses (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner_id TEXT REFERENCES members (id) ON DELETE SET NULL,
    status_id INTEGER NOT NULL REFERENCES project_statuses (id) ON DELETE RESTRICT,
    due_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects (owner_id);
CREATE INDEX IF NOT EXISTS idx_projects_status_id ON projects (status_id);

CREATE TABLE IF NOT EXISTS project_kanban_statuses (
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    color TEXT NOT NULL DEFAULT '#424242',
    PRIMARY KEY (project_id, name)
);

CREATE TABLE IF NOT EXISTS project_kanban_tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    status_name TEXT NOT NULL REFERENCES project_kanban_statuses (name) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    due_at TIMESTAMPTZ,
    assignee_id TEXT REFERENCES members (id) ON DELETE SET NULL,
    parent_task_id TEXT REFERENCES project_kanban_tasks (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_kanban_task_history (
    id BIGSERIAL PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES project_kanban_tasks (id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    performed_by_id TEXT REFERENCES members (id) ON DELETE SET NULL,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    details JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_project_kanban_tasks_project_id ON project_kanban_tasks (project_id);
CREATE INDEX IF NOT EXISTS idx_project_kanban_tasks_status_name ON project_kanban_tasks (status_name);
CREATE INDEX IF NOT EXISTS idx_project_kanban_tasks_assignee_id ON project_kanban_tasks (assignee_id);
