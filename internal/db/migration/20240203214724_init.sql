-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE spinus_user (
	id UUID PRIMARY KEY,
	username VARCHAR(128) UNIQUE NOT NULL CHECK (LENGTH(TRIM(username)) >= 3),
	email VARCHAR(128) UNIQUE NOT NULL CHECK (email ~ '^[a-zA-Z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'),
	password VARCHAR(128) NOT NULL CHECK (LENGTH(password) >= 8)
);

CREATE TYPE energy AS ENUM (
	'electricity',
	'gas',
	'water'
);
CREATE TABLE mm (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	meter_id VARCHAR(64) NOT NULL CHECK (LENGTH(TRIM(meter_id)) >= 3),
	energy ENERGY NOT NULL,
	address VARCHAR(255) NOT NULL CHECK (LENGTH(TRIM(address)) >= 8),
	currency_code VARCHAR(3) NOT NULL CHECK (LENGTH(TRIM(currency_code)) = 3 AND currency_code = UPPER(currency_code)),
	fk_user UUID NOT NULL REFERENCES spinus_user(id)
);

CREATE TABLE sm (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	fk_mm UUID NOT NULL REFERENCES mm(id),
	meter_id VARCHAR(64),
	fin_balance DOUBLE PRECISION NOT NULL,
	fk_user UUID NOT NULL REFERENCES spinus_user(id)
);
CREATE TABLE sm_rdg (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	fk_sm UUID NOT NULL REFERENCES sm(id),
	rdg_val DOUBLE PRECISION NOT NULL,
	rdg_date DATE NOT NULL
);

CREATE TYPE mm_bill_status AS ENUM (
	'in progress',
	'completed'
);
CREATE TABLE mm_bill (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	fk_mm UUID NOT NULL REFERENCES mm(id),
	max_day_diff INT NOT NULL,
	begin_date DATE NOT NULL,
	end_date DATE NOT NULL,
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	from_fin_balance DOUBLE PRECISION NOT NULL,
	to_pay DOUBLE PRECISION NOT NULL,
	status mm_bill_status NOT NULL
);
CREATE TABLE mm_bill_period (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	fk_mm_bill UUID NOT NULL REFERENCES mm_bill(id),
	begin_date DATE NOT NULL,
	end_date DATE NOT NULL,
	begin_rdg_val DOUBLE PRECISION NOT NULL,
	end_rdg_val DOUBLE PRECISION NOT NULL,
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	total_price DOUBLE PRECISION NOT NULL
);
CREATE TYPE sm_bill_status AS ENUM (
	'unpaid',
	'paid'
);
CREATE TABLE sm_bill (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	fk_sm UUID NOT NULL REFERENCES sm(id),
	fk_mm_bill UUID NOT NULL REFERENCES mm_bill(id),
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	from_fin_balance DOUBLE PRECISION NOT NULL,
	to_pay DOUBLE PRECISION NOT NULL,
	status sm_bill_status NOT NULL
);
CREATE TABLE sm_bill_period (
	id UUID PRIMARY KEY,
	created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	fk_sm_bill UUID NOT NULL REFERENCES sm_bill(id),
	fk_mm_bill_period UUID NOT NULL REFERENCES mm_bill_period(id),
	energy_consum DOUBLE PRECISION NOT NULL,
	consum_energy_price DOUBLE PRECISION NOT NULL,
	service_price DOUBLE PRECISION,
	advance_price DOUBLE PRECISION NOT NULL,
	total_price DOUBLE PRECISION NOT NULL
);

-- +goose Down
DROP EXTENSION IF EXISTS "uuid-ossp";
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
