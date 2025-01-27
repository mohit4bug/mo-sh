CREATE TABLE "github_apps" (
	"id" text PRIMARY KEY NOT NULL,
	"slug" text NOT NULL,
	"client_id" text NOT NULL,
	"node_id" text NOT NULL,
	"owner" jsonb NOT NULL,
	"name" text NOT NULL,
	"description" text NOT NULL,
	"external_url" text NOT NULL,
	"html_url" text NOT NULL,
	"created_at" timestamp with time zone NOT NULL,
	"updated_at" timestamp with time zone NOT NULL,
	"permissions" jsonb NOT NULL,
	"events" jsonb NOT NULL,
	"source_id" text NOT NULL
);
--> statement-breakpoint
ALTER TABLE "github_apps" ADD CONSTRAINT "github_apps_source_id_sources_id_fk" FOREIGN KEY ("source_id") REFERENCES "public"."sources"("id") ON DELETE no action ON UPDATE no action;