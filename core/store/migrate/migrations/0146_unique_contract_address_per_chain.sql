-- +goose Up
--- Remove all but most recently added contract_address for each chain. We will no longer allow duplicates, but enforcing that with a db constraint requires CREATE OPERATOR (admin) privilege
-- +goose Down
DROP OPERATOR CLASS IF EXISTS wildcard_cmp USING BTREE CASCADE;
DROP FUNCTION IF EXISTS wildcard_cmp(INTEGER, INTEGER) CASCADE;
