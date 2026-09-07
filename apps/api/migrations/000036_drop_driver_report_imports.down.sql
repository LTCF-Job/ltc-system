CREATE TABLE driver_report_imports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES driver_report_forms(id) ON DELETE CASCADE,
    year_month TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_driver_report_import UNIQUE (form_id, year_month, file_hash)
);

CREATE INDEX idx_driver_report_imports_form_month
    ON driver_report_imports(form_id, year_month);
