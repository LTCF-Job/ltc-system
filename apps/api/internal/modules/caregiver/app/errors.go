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
	// ErrCaregiverInUse 代表照護人員仍被個案關聯（cases.caregiver_id 為 ON DELETE
	// RESTRICT），無法刪除。
	ErrCaregiverInUse = errors.New("caregiver is still referenced by cases")
)
