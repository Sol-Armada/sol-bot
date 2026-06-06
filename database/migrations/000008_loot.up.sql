CREATE TABLE IF NOT EXISTS roll_events (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    attendance_id TEXT REFERENCES attendance(id) ON DELETE SET NULL,
    end_time TIMESTAMPTZ,
    ended BOOLEAN NOT NULL DEFAULT FALSE,
    channel_id TEXT NOT NULL DEFAULT '',
    embed_message_id TEXT NOT NULL DEFAULT '',
    input_message_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_roll_events_attendance_id ON roll_events (attendance_id);
CREATE INDEX IF NOT EXISTS idx_roll_events_ended ON roll_events (ended);
CREATE INDEX IF NOT EXISTS idx_roll_events_created_at ON roll_events (created_at DESC);

CREATE TABLE IF NOT EXISTS roll_items (
    id TEXT PRIMARY KEY,
    roll_event_id TEXT NOT NULL REFERENCES roll_events(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    amount INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    channel_id TEXT,
    message_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_roll_items_roll_event_id ON roll_items (roll_event_id);

CREATE TABLE IF NOT EXISTS roll_entries (
    roll_event_id TEXT NOT NULL REFERENCES roll_events(id) ON DELETE CASCADE,
    roll_item_id TEXT NOT NULL REFERENCES roll_items(id) ON DELETE CASCADE,
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
    choice TEXT NOT NULL CHECK (choice IN ('need', 'greed', 'pass')),
    roll_value INTEGER,
    winner BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (roll_item_id, member_id)
);

CREATE INDEX IF NOT EXISTS idx_roll_entries_roll_event_id ON roll_entries (roll_event_id);
CREATE INDEX IF NOT EXISTS idx_roll_entries_member_id ON roll_entries (member_id);

CREATE TRIGGER trg_notify_roll_events_change
AFTER INSERT OR UPDATE OR DELETE ON roll_events
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_events', 'id');

CREATE TRIGGER trg_notify_roll_entries_change
AFTER INSERT OR UPDATE OR DELETE ON roll_entries
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_events', 'roll_event_id', 'roll_item_id', 'member_id');
