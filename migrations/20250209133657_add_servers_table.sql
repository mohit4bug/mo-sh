-- +goose Up
-- +goose StatementBegin
CREATE TABLE "servers" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" TEXT NOT NULL,
    "hostname" TEXT NOT NULL,
    "port" INTEGER NOT NULL,
    "is_validated" BOOLEAN NOT NULL DEFAULT FALSE,
    "key_id" UUID NOT NULL REFERENCES "keys"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "servers";
-- +goose StatementEnd
