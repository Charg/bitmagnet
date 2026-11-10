-- +goose Up
-- +goose StatementBegin

-- Nothing queries the job payload, but as status is indexed, every job insert and status change had to add a
-- GIN entry for each key and value of the payload (around 100 info hashes per classifier job), and flushing
-- the GIN pending list could stall a single-row statement for close to a minute.
DROP INDEX IF EXISTS queue_jobs_queue_payload_idx;

-- Leading with the primary key made this index useless for anything the primary key index doesn't already do.
DROP INDEX IF EXISTS queue_jobs_id_queue_status_priority_run_after_idx;

-- Finding the next runnable job only ever looks at pending and retry jobs, which are a small fraction of the
-- table as processed jobs are kept for their archival duration.
CREATE INDEX IF NOT EXISTS queue_jobs_runnable_idx ON queue_jobs (queue, priority, run_after)
  WHERE status IN ('pending', 'retry');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS queue_jobs_runnable_idx;
CREATE INDEX IF NOT EXISTS queue_jobs_id_queue_status_priority_run_after_idx
  ON queue_jobs (id, queue, status, priority, run_after);
CREATE INDEX IF NOT EXISTS queue_jobs_queue_payload_idx ON queue_jobs USING gin (queue, payload);

-- +goose StatementEnd
