ALTER TABLE export_job_files
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS file_content,
    DROP COLUMN IF EXISTS storage_path;
