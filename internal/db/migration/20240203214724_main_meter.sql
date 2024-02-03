-- +goose Up
CREATE TABLE main_meter (
  id BIGSERIAL PRIMARY KEY,
  no BIGINT NOT NULL,
  address TEXT NOT NULL
);

-- +goose Down
DROP TABLE main_meter;
