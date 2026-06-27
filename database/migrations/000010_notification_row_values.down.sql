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
