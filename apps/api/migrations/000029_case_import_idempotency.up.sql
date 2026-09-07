CREATE TABLE case_import_idempotency (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_hash TEXT NOT NULL,
    row_key TEXT NOT NULL,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_case_import_idempotency UNIQUE (file_hash, row_key)
);

CREATE INDEX idx_case_import_idempotency_case_id ON case_import_idempotency(case_id);
