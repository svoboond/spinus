-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE spinus_user (
  id INT GENERATED ALWAYS AS IDENTITY,
  username VARCHAR(128) UNIQUE NOT NULL CHECK (LENGTH(TRIM(username)) >= 3),
  email VARCHAR(128) UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'),
  password VARCHAR(255) NOT NULL CHECK (LENGTH(password) >= 8),
  PRIMARY KEY(id)
);

CREATE TABLE main_meter (
  id INT GENERATED ALWAYS AS IDENTITY,
  meter_id VARCHAR(64) NOT NULL CHECK (LENGTH(TRIM(meter_id)) >= 3),
  address VARCHAR(255) NOT NULL CHECK (LENGTH(TRIM(address)) >= 8),
  fk_user INT NOT NULL REFERENCES spinus_user(id),
  PRIMARY KEY(id)
);

-- +goose Down
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS pgcrypto;

DROP TABLE spinus_user;

DROP TABLE main_meter;
