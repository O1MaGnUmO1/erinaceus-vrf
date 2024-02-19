-- +goose Up
ALTER TABLE external_initiators ADD CONSTRAINT "access_key_unique" UNIQUE ("access_key");

ALTER TABLE external_initiators DROP CONSTRAINT "access_key_unique";
