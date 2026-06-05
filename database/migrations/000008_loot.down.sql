DROP TRIGGER IF EXISTS trg_notify_roll_entries_change ON roll_entries;
DROP TRIGGER IF EXISTS trg_notify_roll_events_change ON roll_events;

DROP TABLE IF EXISTS roll_entries;
DROP TABLE IF EXISTS roll_items;
DROP TABLE IF EXISTS roll_events;
