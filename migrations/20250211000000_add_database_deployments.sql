-- +goose Up
-- +goose StatementBegin
CREATE TABLE "database_plans" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "engine" TEXT NOT NULL,
    "version" TEXT NOT NULL,
    "image" TEXT NOT NULL,
    "default_port" INTEGER NOT NULL,
    "is_active" BOOLEAN NOT NULL DEFAULT TRUE,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE ("engine", "version")
);

INSERT INTO "database_plans" ("engine", "version", "image", "default_port")
VALUES ('postgres', '18', 'postgres:18', 5432)
ON CONFLICT ("engine", "version") DO NOTHING;

CREATE TABLE "database_services" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "user_id" UUID NOT NULL REFERENCES "users"("id") ON DELETE CASCADE,
    "server_id" UUID NOT NULL REFERENCES "servers"("id") ON DELETE CASCADE,
    "plan_id" UUID NOT NULL REFERENCES "database_plans"("id"),
    "name" TEXT NOT NULL,
    "database_name" TEXT NOT NULL,
    "database_user" TEXT NOT NULL,
    "database_password" TEXT NOT NULL,
    "container_name" TEXT NOT NULL UNIQUE,
    "volume_name" TEXT NOT NULL UNIQUE,
    "network_name" TEXT NOT NULL DEFAULT 'mo-sh',
    "internal_port" INTEGER NOT NULL DEFAULT 5432,
    "public_port" INTEGER,
    "expose_publicly" BOOLEAN NOT NULL DEFAULT FALSE,
    "status" TEXT NOT NULL DEFAULT 'pending',
    "deployment_logs" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE ("server_id", "name")
);

CREATE TABLE "database_deployment_jobs" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "database_service_id" UUID NOT NULL REFERENCES "database_services"("id") ON DELETE CASCADE,
    "status" TEXT NOT NULL DEFAULT 'pending',
    "error_message" TEXT,
    "started_at" TIMESTAMP,
    "finished_at" TIMESTAMP,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX "database_services_server_public_port_unique"
    ON "database_services" ("server_id", "public_port")
    WHERE "public_port" IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "database_deployment_jobs";
DROP TABLE "database_services";
DROP TABLE "database_plans";
-- +goose StatementEnd
