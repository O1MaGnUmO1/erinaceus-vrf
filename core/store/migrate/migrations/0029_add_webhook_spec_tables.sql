-- +goose Up
CREATE TABLE webhook_specs (
	id SERIAL PRIMARY KEY,
    on_chain_job_spec_id uuid NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL
);

ALTER TABLE jobs ADD COLUMN webhook_spec_id INT REFERENCES webhook_specs(id),
DROP CONSTRAINT chk_only_one_spec,
ADD CONSTRAINT chk_only_one_spec CHECK (
	num_nonnulls(vrf_spec_id, webhook_spec_id) = 1
);

-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT chk_only_one_spec,
ADD CONSTRAINT chk_only_one_spec CHECK (
	num_nonnulls(vrf_spec_id) = 1
);

ALTER TABLE jobs DROP COLUMN webhook_spec_id;

DROP TABLE IF EXISTS webhook_specs;
