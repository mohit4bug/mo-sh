-- +goose Up
-- +goose StatementBegin
CREATE TYPE "public"."docker_installation_status" AS ENUM('in_progress', 'success', 'failure');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE "public"."docker_installation_status";
-- +goose StatementEnd
