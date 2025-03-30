-- +goose Up
-- +goose StatementBegin
CREATE TYPE "public"."source_type" AS ENUM('github');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE "public"."source_type";
-- +goose StatementEnd
