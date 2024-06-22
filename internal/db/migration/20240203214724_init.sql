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
	currency_code VARCHAR(3) NOT NULL CHECK (LENGTH(TRIM(currency_code)) = 3 AND currency_code = UPPER(currency_code)),
	fk_user INT NOT NULL REFERENCES spinus_user(id),
	PRIMARY KEY(id)
);

CREATE TABLE sub_meter (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_main_meter INT NOT NULL REFERENCES main_meter(id),
	subid INT NOT NULL,
	meter_id VARCHAR(64),
	financial_balance DOUBLE PRECISION NOT NULL,
	fk_user INT NOT NULL REFERENCES spinus_user(id),
	PRIMARY KEY(id),
	UNIQUE(fk_main_meter, subid)
);
CREATE TABLE sub_meter_reading (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_sub_meter INT NOT NULL REFERENCES sub_meter(id),
	subid INT NOT NULL,
	reading_value DOUBLE PRECISION NOT NULL,
	reading_date DATE NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_sub_meter, subid),
	UNIQUE(fk_sub_meter, reading_date)
);

CREATE TYPE MAIN_METER_BILLING_STATUS AS ENUM (
	'in progress',
	'completed'
);
CREATE TABLE main_meter_billing (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_main_meter INT NOT NULL REFERENCES main_meter(id),
	subid INT NOT NULL,
	max_day_diff INT NOT NULL,
	begin_date DATE NOT NULL,
	end_date DATE NOT NULL,
	energy_consumption DOUBLE PRECISION NOT NULL,
	consumed_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	from_financial_balance DOUBLE PRECISION NOT NULL,
	to_pay DOUBLE PRECISION NOT NULL,
	status MAIN_METER_BILLING_STATUS NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_main_meter, subid)
);
CREATE TABLE main_meter_billing_period (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_main_billing INT NOT NULL REFERENCES main_meter_billing(id),
	subid INT NOT NULL,
	begin_date DATE NOT NULL,
	end_date DATE NOT NULL,
	begin_reading_value DOUBLE PRECISION NOT NULL,
	end_reading_value DOUBLE PRECISION NOT NULL,
	energy_consumption DOUBLE PRECISION NOT NULL,
	consumed_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	total_price DOUBLE PRECISION NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_main_billing, subid)
);
CREATE TYPE SUB_METER_BILLING_STATUS AS ENUM (
	'unpaid',
	'paid'
);
CREATE TABLE sub_meter_billing (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_sub_meter INT NOT NULL REFERENCES sub_meter(id),
	fk_main_billing INT NOT NULL REFERENCES main_meter_billing(id),
	subid INT NOT NULL,
	energy_consumption DOUBLE PRECISION NOT NULL,
	consumed_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	from_financial_balance DOUBLE PRECISION NOT NULL,
	to_pay DOUBLE PRECISION NOT NULL,
	status SUB_METER_BILLING_STATUS NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_sub_meter, subid)
);
CREATE TABLE sub_meter_billing_period (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_sub_billing INT NOT NULL REFERENCES sub_meter_billing(id),
	fk_main_billing_period INT NOT NULL REFERENCES main_meter_billing_period(id),
	energy_consumption DOUBLE PRECISION NOT NULL,
	consumed_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	total_price DOUBLE PRECISION NOT NULL,
	PRIMARY KEY(id)
);

-- +goose Down
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS pgcrypto;

DROP TABLE spinus_user;

DROP TYPE IF EXISTS energy;
DROP TABLE main_meter;

DROP TABLE sub_meter;

DROP TABLE sub_meter_reading;

DROP TYPE IF EXISTS MAIN_METER_BILLING_STATUS;
DROP TABLE main_meter_billing;
DROP TABLE main_meter_billing_period;
DROP TYPE IF EXISTS SUB_METER_BILLING_STATUS;
DROP TABLE sub_meter_billing;
DROP TABLE sub_meter_billing_period;
