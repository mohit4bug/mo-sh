-- +goose Up
-- +goose StatementBegin
CREATE TABLE "docker_installations" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "server_id" UUID NOT NULL REFERENCES "servers"("id") ON DELETE CASCADE,
    "status" "docker_installation_status" NOT NULL,
    "logs" JSONB NOT NULL,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "docker_installations";
-- +goose StatementEnd
