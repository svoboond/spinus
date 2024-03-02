-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE spinus_user (
	id INT GENERATED ALWAYS AS IDENTITY,
	username VARCHAR(128) UNIQUE NOT NULL CHECK (LENGTH(TRIM(username)) >= 3),
	email VARCHAR(128) UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'),
	password VARCHAR(128) NOT NULL CHECK (LENGTH(password) >= 8),
	PRIMARY KEY(id)
);

CREATE TYPE energy AS ENUM (
	'electricity',
  	'gas',
  	'water'
);
CREATE TABLE main_meter (
	id INT GENERATED ALWAYS AS IDENTITY,
  	meter_id VARCHAR(64) NOT NULL CHECK (LENGTH(TRIM(meter_id)) >= 3),
  	energy ENERGY NOT NULL,
  	address VARCHAR(255) NOT NULL CHECK (LENGTH(TRIM(address)) >= 8),
  	fk_user INT NOT NULL REFERENCES spinus_user(id),
  	PRIMARY KEY(id)
);

CREATE TABLE sub_meter (
  	fk_main_meter INT NOT NULL REFERENCES main_meter(id),
	subid INT NOT NULL,
  	meter_id VARCHAR(64) CHECK (LENGTH(TRIM(meter_id)) >= 3),
  	fk_user INT NOT NULL REFERENCES spinus_user(id),
	PRIMARY KEY(fk_main_meter, subid)
);

-- +goose Down
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS pgcrypto;

DROP TABLE spinus_user;

DROP TYPE IF EXISTS energy;
DROP TABLE main_meter;

DROP TABLE sub_meter;
