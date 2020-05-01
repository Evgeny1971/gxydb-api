alter table gateways
    add column type varchar(16) not null default 'rooms';

alter table gateways alter column type drop default;
