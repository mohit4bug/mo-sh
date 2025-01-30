CREATE TYPE "public"."ssh_key_type" AS ENUM('ed25519', 'rsa');--> statement-breakpoint
ALTER TABLE "private_keys" ADD COLUMN "type" "ssh_key_type" NOT NULL;