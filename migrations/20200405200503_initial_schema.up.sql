CREATE TABLE IF NOT EXISTS users (
    id text PRIMARY KEY,
    display text NOT NULL,
    email text,
    "group" text,
    ip text NOT NULL,
    janus text NOT NULL,
    name text NOT NULL,
    role text NOT NULL,
    system text NOT NULL,
    username text NOT NULL,
    room integer NOT NULL,
    "timestamp" bigint NOT NULL,
    session bigint NOT NULL,
    handle bigint NOT NULL,
    rfid bigint NOT NULL,
    camera boolean DEFAULT false NOT NULL,
    question boolean DEFAULT false NOT NULL,
    self_test boolean DEFAULT false NOT NULL,
    sound_test boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS rooms (
    room integer PRIMARY KEY,
    janus text NOT NULL,
    description text
);
