ALTER TABLE "github_apps" ADD COLUMN "client_secret" text NOT NULL;--> statement-breakpoint
ALTER TABLE "github_apps" ADD COLUMN "webhook_secret" text NOT NULL;--> statement-breakpoint
ALTER TABLE "github_apps" ADD COLUMN "pem" text NOT NULL;