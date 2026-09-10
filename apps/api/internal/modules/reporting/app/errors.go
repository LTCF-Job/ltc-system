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
	// ErrInvalidExportMode 代表匯出模式不在正式支援的白名單內。
	ErrInvalidExportMode = errors.New("invalid export mode")
	// ErrCaseIDsRequired 代表匯出範圍未明確指定個案。
	ErrCaseIDsRequired = errors.New("case ids are required")
	// ErrInvalidPeriodYM 申報年月格式不是民國 5 碼（例如 11507）。
	ErrInvalidPeriodYM = errors.New("invalid ROC period, expected RRRMM")
	// ErrNoExportData 指定條件下一份申報檔都產不出來。
	//
	// 這與「某些欄位缺資料」是兩回事：缺欄位一律留白照樣出檔（不阻擋原則），
	// 但整批連一列都組不出來時如果還回成功，使用者只會看到「已產生 0 份」配一張
	// 空表格，完全看不出是月份選錯、個案在待維護，還是根本沒有搭乘紀錄。
	ErrNoExportData = errors.New("no claimable data in the given scope")
	// ErrRegionsRequired 依區域批次匯出時未指定任何區域。
	ErrRegionsRequired = errors.New("regions are required")
	// ErrSiteIDsRequired 依據點匯出時未指定任何據點。
	ErrSiteIDsRequired = errors.New("site ids are required")
	// ErrPeriodsRequired 未指定任何申報月份。
	ErrPeriodsRequired = errors.New("period ym list is required")
)
