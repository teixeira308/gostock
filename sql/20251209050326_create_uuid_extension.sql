-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- +goose Down
-- SELECT 1; -- Do nothing or consider dropping if truly needed
