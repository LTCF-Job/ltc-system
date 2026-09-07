package app

import "errors"

var (
	// ErrCaregiverNotFound 代表查無照護人員資料。
	ErrCaregiverNotFound = errors.New("caregiver not found")
	// ErrCaregiverNameRequired 代表未提供照護人員姓名。
	ErrCaregiverNameRequired = errors.New("caregiver name is required")
	// ErrCaregiverTypeInvalid 代表未提供或提供了非既定選項的照護人員類型。
	ErrCaregiverTypeInvalid = errors.New("caregiver type must be case_manager or specialist")
	// ErrCaregiverStatusInvalid 代表照護人員狀態不在 active／inactive 允許值內。
	ErrCaregiverStatusInvalid = errors.New("caregiver status must be active or inactive")
	// ErrCaregiverSiteNotFound 代表匯入時以名稱查無單位；這是可人工關聯的警告，不是查詢故障。
	ErrCaregiverSiteNotFound = errors.New("caregiver site not found")
)
