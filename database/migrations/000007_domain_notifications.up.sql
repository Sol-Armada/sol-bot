CREATE OR REPLACE FUNCTION notify_domain_change()
RETURNS TRIGGER AS $$
DECLARE
    channel_name TEXT := TG_ARGV[0];
    pk_columns TEXT[] := TG_ARGV[1:TG_NARGS-1];
    row_old JSONB := CASE WHEN TG_OP IN ('UPDATE', 'DELETE') THEN to_jsonb(OLD) ELSE '{}'::JSONB END;
    row_new JSONB := CASE WHEN TG_OP IN ('INSERT', 'UPDATE') THEN to_jsonb(NEW) ELSE '{}'::JSONB END;
    row_data JSONB := CASE WHEN TG_OP = 'DELETE' THEN row_old ELSE row_new END;
    primary_key JSONB := '{}'::JSONB;
    changed_columns TEXT[] := ARRAY[]::TEXT[];
    col TEXT;
BEGIN
    FOREACH col IN ARRAY pk_columns LOOP
        primary_key := primary_key || jsonb_build_object(col, row_data -> col);
    END LOOP;

    IF TG_OP = 'UPDATE' THEN
        SELECT COALESCE(array_agg(COALESCE(new_item.key, old_item.key) ORDER BY COALESCE(new_item.key, old_item.key)), ARRAY[]::TEXT[])
        INTO changed_columns
        FROM jsonb_each(row_new) AS new_item
        FULL OUTER JOIN jsonb_each(row_old) AS old_item ON new_item.key = old_item.key
        WHERE new_item.value IS DISTINCT FROM old_item.value;
    END IF;

    PERFORM pg_notify(
        channel_name,
        jsonb_build_object(
            'payload_version', 1,
            'operation', TG_OP,
            'schema', TG_TABLE_SCHEMA,
            'table', TG_TABLE_NAME,
            'primary_key', primary_key,
            'occurred_at', NOW(),
            'changed_columns', CASE WHEN TG_OP = 'UPDATE' THEN to_jsonb(changed_columns) ELSE '[]'::JSONB END
        )::TEXT
    );

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notify_members_change
AFTER INSERT OR UPDATE OR DELETE ON members
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_members', 'id');

CREATE TRIGGER trg_notify_member_blueprints_change
AFTER INSERT OR UPDATE OR DELETE ON member_blueprints
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_members', 'member_id', 'blueprint_id');

CREATE TRIGGER trg_notify_attendance_change
AFTER INSERT OR UPDATE OR DELETE ON attendance
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_attendance', 'id');

CREATE TRIGGER trg_notify_attendance_payouts_change
AFTER INSERT OR UPDATE OR DELETE ON attendance_payouts
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_attendance', 'attendance_id');

CREATE TRIGGER trg_notify_attendance_participants_change
AFTER INSERT OR UPDATE OR DELETE ON attendance_participants
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_attendance', 'attendance_id', 'member_id');

CREATE TRIGGER trg_notify_attendance_tags_change
AFTER INSERT OR UPDATE OR DELETE ON attendance_tags
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_attendance', 'tag');

CREATE TRIGGER trg_notify_attendance_names_change
AFTER INSERT OR UPDATE OR DELETE ON attendance_names
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_attendance', 'name');

CREATE TRIGGER trg_notify_tokens_change
AFTER INSERT OR UPDATE OR DELETE ON tokens
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_tokens', 'id');

CREATE TRIGGER trg_notify_giveaways_change
AFTER INSERT OR UPDATE OR DELETE ON giveaways
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_events', 'id');

CREATE TRIGGER trg_notify_raffles_change
AFTER INSERT OR UPDATE OR DELETE ON raffles
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_events', 'id');

CREATE TRIGGER trg_notify_sos_tickets_change
AFTER INSERT OR UPDATE OR DELETE ON sos_tickets
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_docs', 'id');

CREATE TRIGGER trg_notify_kanban_cards_change
AFTER INSERT OR UPDATE OR DELETE ON kanban_cards
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_docs', 'id');

CREATE TRIGGER trg_notify_blueprint_docs_change
AFTER INSERT OR UPDATE OR DELETE ON blueprint_docs
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_docs', 'id');

CREATE TRIGGER trg_notify_rsi_info_change
AFTER INSERT OR UPDATE OR DELETE ON rsi_info
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_rsi', 'handle');

CREATE TRIGGER trg_notify_activity_logs_change
AFTER INSERT OR UPDATE OR DELETE ON activity_logs
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_logs', 'id');

CREATE TRIGGER trg_notify_command_logs_change
AFTER INSERT OR UPDATE OR DELETE ON command_logs
FOR EACH ROW
EXECUTE FUNCTION notify_domain_change('solbot_logs', 'id');
