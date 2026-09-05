ALTER TABLE ride_records
    ADD COLUMN based_on_fingerprint TEXT;

COMMENT ON COLUMN ride_records.based_on_fingerprint IS
    '人工更正當下所依據的 ride_sources 快照；來源變更後不得沿用舊更正';
