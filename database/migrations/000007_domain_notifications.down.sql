DROP TRIGGER IF EXISTS trg_notify_members_change ON members;
DROP TRIGGER IF EXISTS trg_notify_member_blueprints_change ON member_blueprints;
DROP TRIGGER IF EXISTS trg_notify_attendance_change ON attendance;
DROP TRIGGER IF EXISTS trg_notify_attendance_payouts_change ON attendance_payouts;
DROP TRIGGER IF EXISTS trg_notify_attendance_participants_change ON attendance_participants;
DROP TRIGGER IF EXISTS trg_notify_attendance_tags_change ON attendance_tags;
DROP TRIGGER IF EXISTS trg_notify_attendance_names_change ON attendance_names;
DROP TRIGGER IF EXISTS trg_notify_tokens_change ON tokens;
DROP TRIGGER IF EXISTS trg_notify_giveaways_change ON giveaways;
DROP TRIGGER IF EXISTS trg_notify_raffles_change ON raffles;
DROP TRIGGER IF EXISTS trg_notify_sos_tickets_change ON sos_tickets;
DROP TRIGGER IF EXISTS trg_notify_kanban_cards_change ON kanban_cards;
DROP TRIGGER IF EXISTS trg_notify_blueprint_docs_change ON blueprint_docs;
DROP TRIGGER IF EXISTS trg_notify_rsi_info_change ON rsi_info;
DROP TRIGGER IF EXISTS trg_notify_activity_logs_change ON activity_logs;
DROP TRIGGER IF EXISTS trg_notify_command_logs_change ON command_logs;

DROP FUNCTION IF EXISTS notify_domain_change();
