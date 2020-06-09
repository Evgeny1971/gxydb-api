ALTER TABLE gateways
    ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE users
    ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE rooms
    ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE sessions
    ALTER COLUMN created_at SET DEFAULT now();

DROP FUNCTION IF EXISTS now_utc();
