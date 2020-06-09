DROP FUNCTION IF EXISTS now_utc();

CREATE FUNCTION now_utc()
    RETURNS TIMESTAMP AS
$$
SELECT now() AT TIME ZONE 'utc';
$$ LANGUAGE SQL;

ALTER TABLE gateways
    ALTER COLUMN created_at SET DEFAULT now_utc();
ALTER TABLE users
    ALTER COLUMN created_at SET DEFAULT now_utc();
ALTER TABLE rooms
    ALTER COLUMN created_at SET DEFAULT now_utc();
ALTER TABLE sessions
    ALTER COLUMN created_at SET DEFAULT now_utc();
