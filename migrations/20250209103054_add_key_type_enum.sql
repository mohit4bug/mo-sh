-- +goose Up
-- +goose StatementBegin
CREATE TYPE "public"."key_type" AS ENUM('ed25519', 'rsa');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE "public"."key_type";
-- +goose StatementEnd
