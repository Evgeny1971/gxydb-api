DROP TABLE IF EXISTS dynamic_config;
CREATE TABLE IF NOT EXISTS dynamic_config
(
    id         BIGSERIAL PRIMARY KEY,
    key        VARCHAR(255) UNIQUE      NOT NULL,
    value      TEXT                     NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS dynamic_config_key_updated_at_idx
    ON dynamic_config USING BTREE (key, updated_at);

CREATE INDEX IF NOT EXISTS dynamic_config_updated_at_key_idx
    ON dynamic_config USING BTREE (updated_at, key);
