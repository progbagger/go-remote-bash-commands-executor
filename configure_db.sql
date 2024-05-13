CREATE TABLE IF NOT EXISTS commands (
    id SERIAL PRIMARY KEY,
    command TEXT NOT NULL
);

DROP TYPE IF EXISTS env_entry CASCADE;

CREATE TYPE env_entry AS (key TEXT, value TEXT);

CREATE TABLE IF NOT EXISTS inputs (
    id SERIAL REFERENCES commands (id),
    input TEXT,
    args TEXT ARRAY,
    env env_entry ARRAY
);

CREATE TABLE IF NOT EXISTS outputs (
    id SERIAL REFERENCES commands (id),
    output TEXT,
    errors TEXT
);

CREATE TABLE IF NOT EXISTS statuses (
    id SERIAL REFERENCES commands (id),
    exit_code INTEGER,
    status TEXT NOT NULL
);

CREATE OR REPLACE FUNCTION outputs_statuses_trigger_fnc()
RETURNS trigger AS
$$
BEGIN
INSERT INTO outputs (id) VALUES (
  NEW.id
);
INSERT INTO statuses (id, status) VALUES (
  NEW.id,
  'RUNNING'
);
RETURN NEW;
END;
$$ LANGUAGE "plpgsql";

CREATE
OR
REPLACE
    TRIGGER after_command_insert AFTER
INSERT
    ON commands FOR EACH ROW
EXECUTE PROCEDURE outputs_statuses_trigger_fnc ();
