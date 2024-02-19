-- +goose Up
DROP TABLE evm_chains CASCADE;


-- +goose Down
-- evm_chains definition
CREATE TABLE evm_chains (
    id numeric(78) NOT NULL,
    cfg jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    enabled bool NOT NULL DEFAULT true,
    CONSTRAINT evm_chains_pkey PRIMARY KEY (id)
);

-- evm_chains foreign keys
ALTER TABLE evm_log_poller_filters ADD CONSTRAINT evm_log_poller_filters_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) DEFERRABLE;
ALTER TABLE evm_log_poller_blocks ADD CONSTRAINT evm_log_poller_blocks_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE log_broadcasts ADD CONSTRAINT log_broadcasts_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE evm_logs ADD CONSTRAINT evm_logs_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE vrf_specs ADD CONSTRAINT vrf_specs_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE evm_heads ADD CONSTRAINT heads_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE evm_forwarders ADD CONSTRAINT evm_forwarders_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE blockhash_store_specs ADD CONSTRAINT blockhash_store_specs_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE evm_key_states ADD CONSTRAINT eth_key_states_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE log_broadcasts_pending ADD CONSTRAINT log_broadcasts_pending_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;
ALTER TABLE eth_txes ADD CONSTRAINT eth_txes_evm_chain_id_fkey FOREIGN KEY (evm_chain_id) REFERENCES evm_chains(id) ON DELETE CASCADE DEFERRABLE NOT VALID;

