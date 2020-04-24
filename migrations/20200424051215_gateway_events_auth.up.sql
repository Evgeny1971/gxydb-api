alter table gateways
    add column events_password char(60) not null default '';
