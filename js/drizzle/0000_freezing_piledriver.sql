CREATE TYPE "public"."source_type" AS ENUM('github');--> statement-breakpoint
CREATE TABLE "sources" (
	"id" text PRIMARY KEY NOT NULL,
	"name" text NOT NULL,
	"type" "source_type" NOT NULL
);
