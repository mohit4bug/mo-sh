-- +goose Up
-- +goose StatementBegin
ALTER TABLE "servers"
    ADD COLUMN "has_caddy" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN "caddy_installation_logs" JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN "is_caddy_installation_task_running" BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "servers"
    DROP COLUMN "is_caddy_installation_task_running",
    DROP COLUMN "caddy_installation_logs",
    DROP COLUMN "has_caddy";
-- +goose StatementEnd
