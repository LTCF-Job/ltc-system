package app

import "errors"

var (
	// ErrPrecheckBlocked 前置檢核存在阻斷性錯誤，不建立匯出工作。
	ErrPrecheckBlocked = errors.New("precheck blocked the export")
	// ErrExportJobNotFound 查無指定的匯出工作。
	ErrExportJobNotFound = errors.New("export job not found")
	// ErrExportFileNotFound 該匯出工作沒有指定個案的檔案。
	ErrExportFileNotFound = errors.New("export job file not found")
	// ErrNotZipJob 對非壓縮檔模式的工作要求整包下載。
	ErrNotZipJob = errors.New("export job is not a zip job")
	// ErrNoClaimRows 指定條件下沒有任何可申報的資料列。
	ErrNoClaimRows = errors.New("no claimable rows for the given filters")
	// ErrInvalidPeriodYM 申報年月格式不是民國 5 碼（例如 11507）。
	ErrInvalidPeriodYM = errors.New("invalid ROC period, expected RRRMM")
)
