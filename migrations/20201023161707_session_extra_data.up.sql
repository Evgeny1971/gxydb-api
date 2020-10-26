alter table sessions
    add column extra jsonb null,
    add column gateway_handle_textroom bigint null;
