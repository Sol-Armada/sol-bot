CREATE TABLE IF NOT EXISTS schema_migrations (
	id SERIAL PRIMARY KEY,
	name TEXT UNIQUE NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_name ON schema_migrations (name);
CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations (applied_at DESC);
