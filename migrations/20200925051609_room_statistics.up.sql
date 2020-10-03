DROP TABLE IF EXISTS room_statistics;
CREATE TABLE IF NOT EXISTS room_statistics
(
    room_id BIGINT REFERENCES rooms PRIMARY KEY NOT NULL,
    on_air  INTEGER                             NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS room_statistics_room_id_idx
    ON room_statistics USING BTREE (room_id);
