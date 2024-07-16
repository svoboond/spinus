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
CREATE TABLE mm (
	id INT GENERATED ALWAYS AS IDENTITY,
	meter_id VARCHAR(64) NOT NULL CHECK (LENGTH(TRIM(meter_id)) >= 3),
	energy ENERGY NOT NULL,
	address VARCHAR(255) NOT NULL CHECK (LENGTH(TRIM(address)) >= 8),
	currency_code VARCHAR(3) NOT NULL CHECK (LENGTH(TRIM(currency_code)) = 3 AND currency_code = UPPER(currency_code)),
	fk_user INT NOT NULL REFERENCES spinus_user(id),
	PRIMARY KEY(id)
);

CREATE TABLE sm (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_mm INT NOT NULL REFERENCES mm(id),
	subid INT NOT NULL,
	meter_id VARCHAR(64),
	fin_balance DOUBLE PRECISION NOT NULL,
	fk_user INT NOT NULL REFERENCES spinus_user(id),
	PRIMARY KEY(id),
	UNIQUE(fk_mm, subid)
);
CREATE TABLE sm_rdg (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_sm INT NOT NULL REFERENCES sm(id),
	subid INT NOT NULL,
	rdg_val DOUBLE PRECISION NOT NULL,
	rdg_date DATE NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_sm, subid),
	UNIQUE(fk_sm, rdg_date)
);

CREATE TYPE mm_bill_status AS ENUM (
	'in progress',
	'completed'
);
CREATE TABLE mm_bill (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_mm INT NOT NULL REFERENCES mm(id),
	subid INT NOT NULL,
	max_day_diff INT NOT NULL,
	begin_date DATE NOT NULL,
	end_date DATE NOT NULL,
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	from_fin_balance DOUBLE PRECISION NOT NULL,
	to_pay DOUBLE PRECISION NOT NULL,
	status mm_bill_status NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_mm, subid)
);
CREATE TABLE mm_bill_period (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_mm_bill INT NOT NULL REFERENCES mm_bill(id),
	subid INT NOT NULL,
	begin_date DATE NOT NULL,
	end_date DATE NOT NULL,
	begin_rdg_val DOUBLE PRECISION NOT NULL,
	end_rdg_val DOUBLE PRECISION NOT NULL,
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	total_price DOUBLE PRECISION NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_mm_bill, subid)
);
CREATE TYPE sm_bill_status AS ENUM (
	'unpaid',
	'paid'
);
CREATE TABLE sm_bill (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_sm INT NOT NULL REFERENCES sm(id),
	fk_mm_bill INT NOT NULL REFERENCES mm_bill(id),
	subid INT NOT NULL,
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	from_fin_balance DOUBLE PRECISION NOT NULL,
	to_pay DOUBLE PRECISION NOT NULL,
	status sm_bill_status NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(fk_sm, subid)
);
CREATE TABLE sm_bill_period (
	id INT GENERATED ALWAYS AS IDENTITY,
	fk_sm_bill INT NOT NULL REFERENCES sm_bill(id),
	fk_mm_bill_period INT NOT NULL REFERENCES mm_bill_period(id),
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
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
DROP TABLE mm;

DROP TABLE sm;

DROP TABLE sm_rdg;

DROP TYPE IF EXISTS mm_bill_status;
DROP TABLE mm_bill;
DROP TABLE mm_bill_period;
DROP TYPE IF EXISTS sm_bill_status;
DROP TABLE sm_bill;
DROP TABLE sm_bill_period;
