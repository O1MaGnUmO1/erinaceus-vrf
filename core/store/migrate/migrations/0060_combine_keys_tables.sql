-- +goose Up
CREATE TABLE encrypted_key_rings(
    encrypted_keys jsonb,
    updated_at timestamptz NOT NULL
);

CREATE TABLE eth_key_states(
	id SERIAL PRIMARY KEY,
	address bytea UNIQUE NOT NULL,
	next_nonce bigint NOT NULL DEFAULT 0,
	is_funding boolean DEFAULT false NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	CONSTRAINT chk_address_length CHECK ((octet_length(address) = 20))
);

ALTER TABLE eth_txes DROP CONSTRAINT eth_txes_from_address_fkey;
-- Need the NOT VALID constraint here because the eth_key_states are not created yet; they will be created on application boot
ALTER TABLE eth_txes ADD CONSTRAINT eth_txes_from_address_fkey FOREIGN KEY (from_address) REFERENCES eth_key_states(address) NOT VALID;
ALTER TABLE vrf_specs DROP CONSTRAINT vrf_specs_public_key_fkey;

-- +goose Down
DROP TABLE encrypted_key_rings;
ALTER TABLE eth_txes DROP CONSTRAINT eth_txes_from_address_fkey;
DROP TABLE eth_key_states;
ALTER TABLE eth_txes ADD CONSTRAINT eth_txes_from_address_fkey FOREIGN KEY (from_address) REFERENCES keys(address);
ALTER TABLE vrf_specs ADD CONSTRAINT vrf_specs_public_key_fkey FOREIGN KEY (public_key) REFERENCES encrypted_vrf_keys(public_key) ON DELETE CASCADE DEFERRABLE INITIALLY IMMEDIATE;
