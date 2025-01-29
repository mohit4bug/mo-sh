CREATE TABLE "servers" (
	"id" text PRIMARY KEY NOT NULL,
	"name" text NOT NULL,
	"hostname" text NOT NULL,
	"port" integer NOT NULL,
	"private_key_id" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone NOT NULL
);
--> statement-breakpoint
ALTER TABLE "servers" ADD CONSTRAINT "servers_private_key_id_private_keys_id_fk" FOREIGN KEY ("private_key_id") REFERENCES "public"."private_keys"("id") ON DELETE no action ON UPDATE no action;