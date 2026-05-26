CREATE TABLE members (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL DEFAULT '',
	rank INTEGER NOT NULL DEFAULT 0,
	joined TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated TIMESTAMPTZ,
	is_bot BOOLEAN NOT NULL DEFAULT FALSE,
	is_ally BOOLEAN NOT NULL DEFAULT FALSE,
	is_affiliate BOOLEAN NOT NULL DEFAULT FALSE,
	is_guest BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_members_rank ON members (rank);
CREATE INDEX idx_members_is_bot ON members (is_bot);

CREATE TABLE member_blueprints (
	member_id TEXT NOT NULL REFERENCES members (id) ON DELETE CASCADE,
	blueprint_id TEXT NOT NULL,
	PRIMARY KEY (member_id, blueprint_id)
);

CREATE INDEX idx_member_blueprints_blueprint_id ON member_blueprints (blueprint_id);

CREATE TABLE attendance (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	submitted_by TEXT REFERENCES members (id) ON DELETE SET NULL,
	recorded BOOLEAN NOT NULL DEFAULT FALSE,
	successful BOOLEAN NOT NULL DEFAULT FALSE,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	tokenable BOOLEAN NOT NULL DEFAULT FALSE,
	status TEXT NOT NULL DEFAULT 'active',
	channel_id TEXT NOT NULL DEFAULT '',
	message_id TEXT NOT NULL DEFAULT '',
	date_created TIMESTAMPTZ NOT NULL,
	date_updated TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_attendance_date_created ON attendance (date_created DESC);
CREATE INDEX idx_attendance_recorded_status ON attendance (recorded, status);

CREATE TABLE attendance_payouts (
	attendance_id TEXT PRIMARY KEY REFERENCES attendance (id) ON DELETE CASCADE,
	total BIGINT NOT NULL DEFAULT 0,
	per_member BIGINT NOT NULL DEFAULT 0,
	org_take BIGINT NOT NULL DEFAULT 0,
	date_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE attendance_participants (
	attendance_id TEXT NOT NULL REFERENCES attendance (id) ON DELETE CASCADE,
	member_id TEXT NOT NULL REFERENCES members (id) ON DELETE RESTRICT,
	joined_at_start BOOLEAN NOT NULL DEFAULT FALSE,
	stayed_until_end BOOLEAN NOT NULL DEFAULT FALSE,
	has_issue BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (attendance_id, member_id)
);

CREATE INDEX idx_attendance_participants_member_id ON attendance_participants (member_id);
CREATE INDEX idx_attendance_participants_stayed ON attendance_participants (attendance_id, stayed_until_end);
CREATE INDEX idx_attendance_participants_issues ON attendance_participants (attendance_id, has_issue);

CREATE TABLE attendance_tags (
	tag TEXT PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE attendance_names (
	name TEXT PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tokens (
	id TEXT PRIMARY KEY,
	member_id TEXT NOT NULL REFERENCES members (id) ON DELETE RESTRICT,
	amount INTEGER NOT NULL,
	reason TEXT NOT NULL,
	attendance_id TEXT REFERENCES attendance (id) ON DELETE SET NULL,
	comment TEXT,
	giver_id TEXT REFERENCES members (id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tokens_member_id ON tokens (member_id);
CREATE INDEX idx_tokens_attendance_id ON tokens (attendance_id);
CREATE INDEX idx_tokens_created_at ON tokens (created_at DESC);

CREATE TABLE activity_logs (
	id BIGSERIAL PRIMARY KEY,
	who_id TEXT REFERENCES members (id) ON DELETE SET NULL,
	occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	what TEXT NOT NULL,
	where_text TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_activity_logs_who_id ON activity_logs (who_id);
CREATE INDEX idx_activity_logs_occurred_at ON activity_logs (occurred_at DESC);

CREATE TABLE command_logs (
	id BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	user_id TEXT NOT NULL DEFAULT '',
	interaction_type INTEGER NOT NULL DEFAULT 0,
	button_id TEXT NOT NULL DEFAULT '',
	error_text TEXT NOT NULL DEFAULT '',
	options_json TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX idx_command_logs_occurred_at ON command_logs (occurred_at DESC);
CREATE INDEX idx_command_logs_name_occurred_at ON command_logs (name, occurred_at DESC);

CREATE TABLE sos_tickets (
	id TEXT PRIMARY KEY,
	member_id TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT '{}',
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sos_tickets_member_id ON sos_tickets (member_id);

CREATE TABLE kanban_cards (
	id TEXT PRIMARY KEY,
	payload_json TEXT NOT NULL DEFAULT '{}',
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE blueprint_docs (
	id TEXT PRIMARY KEY,
	payload_json TEXT NOT NULL DEFAULT '{}',
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE giveaways (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	items_json TEXT NOT NULL DEFAULT '{}',
	attendance_id TEXT REFERENCES attendance (id) ON DELETE SET NULL,
	end_time TIMESTAMPTZ,
	ended BOOLEAN NOT NULL DEFAULT FALSE,
	channel_id TEXT NOT NULL DEFAULT '',
	embed_message_id TEXT NOT NULL DEFAULT '',
	input_message_id TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_giveaways_attendance_id ON giveaways (attendance_id);
CREATE INDEX idx_giveaways_ended ON giveaways (ended);

CREATE TABLE raffles (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	attendance_id TEXT REFERENCES attendance (id) ON DELETE SET NULL,
	prize TEXT NOT NULL,
	quantity INTEGER NOT NULL DEFAULT 1,
	tickets_json TEXT NOT NULL DEFAULT '{}',
	winners TEXT[] NOT NULL DEFAULT '{}',
	ended BOOLEAN NOT NULL DEFAULT FALSE,
	test BOOLEAN NOT NULL DEFAULT FALSE,
	channel_id TEXT NOT NULL DEFAULT '',
	message_id TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_raffles_attendance_id ON raffles (attendance_id);
CREATE INDEX idx_raffles_created_at ON raffles (created_at DESC);
