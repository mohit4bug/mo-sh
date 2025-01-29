ALTER TABLE "github_apps" RENAME COLUMN "pem" TO "private_key_id";--> statement-breakpoint
ALTER TABLE "private_keys" ALTER COLUMN "updated_at" SET DEFAULT now();--> statement-breakpoint
ALTER TABLE "servers" ALTER COLUMN "updated_at" SET DEFAULT now();--> statement-breakpoint
ALTER TABLE "sources" ALTER COLUMN "updated_at" SET DEFAULT now();--> statement-breakpoint
ALTER TABLE "github_apps" ADD CONSTRAINT "github_apps_private_key_id_private_keys_id_fk" FOREIGN KEY ("private_key_id") REFERENCES "public"."private_keys"("id") ON DELETE no action ON UPDATE no action;