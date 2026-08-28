ALTER TABLE holidays
    ADD COLUMN is_day_off BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN holidays.is_day_off IS 'TRUE for leave days, FALSE for government-designated make-up workdays';
