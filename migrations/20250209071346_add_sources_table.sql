-- +goose Up
-- +goose StatementBegin
CREATE TABLE "sources" (
	"id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	"name" text NOT NULL,
	"type" "source_type" NOT NULL,
	"has_github_app" BOOLEAN NOT NULL DEFAULT false,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "sources";
-- +goose StatementEnd
