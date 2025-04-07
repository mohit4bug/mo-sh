-- +goose Up
-- +goose StatementBegin
CREATE TABLE "github_apps" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	"slug" TEXT NOT NULL,
	"client_id" TEXT NOT NULL,
	"node_id" TEXT NOT NULL,
	"owner" JSONB NOT NULL,
	"name" TEXT NOT NULL,
	"description" TEXT NOT NULL,
	"external_url" TEXT NOT NULL,
	"html_url" TEXT NOT NULL,
	"created_at" TIMESTAMP NOT NULL,
	"updated_at" TIMESTAMP NOT NULL,
	"permissions" JSONB NOT NULL,
	"events" JSONB NOT NULL,
	"source_id" UUID NOT NULL REFERENCES "sources"("id") ON DELETE CASCADE,
    "client_secret" TEXT NOT NULL,
    "webhook_secret" TEXT NOT NULL,
    "key_id" UUID NOT NULL REFERENCES "keys"("id") ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "github_apps";
-- +goose StatementEnd
