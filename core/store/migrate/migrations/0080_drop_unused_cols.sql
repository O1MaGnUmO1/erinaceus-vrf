-- +goose Up
ALTER TABLE jobs ADD COLUMN created_at timestamptz;

UPDATE jobs SET created_at=vrf_specs.created_at FROM vrf_specs WHERE jobs.vrf_spec_id = vrf_specs.id;
UPDATE jobs SET created_at=webhook_specs.created_at FROM webhook_specs WHERE jobs.webhook_spec_id = webhook_specs.id;

UPDATE jobs SET created_at = NOW() WHERE created_at IS NULL;
CREATE INDEX idx_jobs_created_at ON jobs USING BRIN (created_at);
ALTER TABLE jobs ALTER COLUMN created_at SET NOT NULL;

-- +goose Down
ALTER TABLE jobs DROP COLUMN created_at;
