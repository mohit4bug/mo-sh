-- +goose Up
-- +goose StatementBegin
CREATE TYPE "public"."installation_status" AS ENUM (
    'not_started',
    'in_progress',
    'completed',
    'failed',
    'cancelled'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE "public"."installation_status";
-- +goose StatementEnd
