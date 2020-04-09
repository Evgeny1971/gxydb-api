---------------
-- Functions --
---------------

DROP FUNCTION IF EXISTS now_utc();

CREATE FUNCTION now_utc()
    RETURNS TIMESTAMP AS
$$
SELECT now() AT TIME ZONE 'utc';
$$ LANGUAGE SQL;

------------
-- Tables --
------------

DROP TABLE IF EXISTS gateways;
CREATE TABLE IF NOT EXISTS gateways
(
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(16) UNIQUE                         NOT NULL,
    description    VARCHAR(255)                               NULL,
    url            VARCHAR(1024)                              NOT NULL,
    admin_url      VARCHAR(1024)                              NOT NULL,
    admin_password TEXT                                       NOT NULL,
    disabled       BOOLEAN                  DEFAULT FALSE     NOT NULL,
    properties     JSONB                                      NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT now_utc() NOT NULL,
    updated_at     TIMESTAMP WITH TIME ZONE                   NULL,
    removed_at     TIMESTAMP WITH TIME ZONE                   NULL
);

DROP TABLE IF EXISTS users;
CREATE TABLE IF NOT EXISTS users
(
    id          BIGSERIAL PRIMARY KEY,
    accounts_id VARCHAR(36) UNIQUE                         NOT NULL,
    email       VARCHAR(255)                               NULL,
    first_name  VARCHAR(255)                               NULL,
    last_name   VARCHAR(255)                               NULL,
    username    VARCHAR(255)                               NULL,
    disabled    BOOLEAN                  DEFAULT FALSE     NOT NULL,
    properties  JSONB                                      NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT now_utc() NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE                   NULL,
    removed_at  TIMESTAMP WITH TIME ZONE                   NULL
);

DROP TABLE IF EXISTS rooms;
CREATE TABLE IF NOT EXISTS rooms
(
    id                 BIGSERIAL PRIMARY KEY,
    name               VARCHAR(64) UNIQUE                         NOT NULL,
    default_gateway_id BIGINT REFERENCES gateways                 NOT NULL,
    gateway_uid        INTEGER UNIQUE                             NOT NULL,
    secret             TEXT                                       NOT NULL,
    disabled           BOOLEAN                  DEFAULT FALSE     NOT NULL,
    properties         JSONB                                      NULL,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT now_utc() NOT NULL,
    updated_at         TIMESTAMP WITH TIME ZONE                   NULL,
    removed_at         TIMESTAMP WITH TIME ZONE                   NULL,
    UNIQUE (default_gateway_id, gateway_uid)
);

DROP TABLE IF EXISTS sessions;
CREATE TABLE IF NOT EXISTS sessions
(
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT REFERENCES users                    NOT NULL,
    room_id         BIGINT REFERENCES rooms                    NULL,

    gateway_id      BIGINT REFERENCES gateways                 NULL,
    gateway_session BIGINT                                     NULL,
    gateway_handle  BIGINT                                     NULL,
    gateway_feed    BIGINT                                     NULL,

    display         VARCHAR(512)                               NULL,
    camera          BOOLEAN                  DEFAULT FALSE     NOT NULL,
    question        BOOLEAN                  DEFAULT FALSE     NOT NULL,
    self_test       BOOLEAN                  DEFAULT FALSE     NOT NULL,
    sound_test      BOOLEAN                  DEFAULT FALSE     NOT NULL,
    user_agent      TEXT                                       NULL,
    ip_address      INET                                       NULL,
    properties      JSONB                                      NULL,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT now_utc() NOT NULL,
    updated_at      TIMESTAMP WITH TIME ZONE                   NULL,
    removed_at      TIMESTAMP WITH TIME ZONE                   NULL,
    UNIQUE (gateway_id, gateway_session)
);

DROP TABLE IF EXISTS composites;
CREATE TABLE IF NOT EXISTS composites
(
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(16) UNIQUE NOT NULL,
    description VARCHAR(255)       NULL
);

DROP TABLE IF EXISTS composites_rooms;
CREATE TABLE IF NOT EXISTS composites_rooms
(
    composite_id BIGINT REFERENCES composites NOT NULL,
    room_id      BIGINT REFERENCES rooms      NOT NULL,
    gateway_id   BIGINT REFERENCES gateways   NOT NULL,
    position     INTEGER DEFAULT 1            NOT NULL,
    PRIMARY KEY (composite_id, room_id, gateway_id)
);

-- TODO: program state (shidur)

-------------
-- Indexes --
-------------

CREATE INDEX IF NOT EXISTS sessions_room_id_idx
    ON sessions USING BTREE (room_id);


