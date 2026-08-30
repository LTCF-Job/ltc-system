package app

import "errors"

var (
	// ErrCaregiverNotFound 代表查無照護人員資料。
	ErrCaregiverNotFound = errors.New("caregiver not found")
	// ErrCaregiverNameRequired 代表未提供照護人員姓名。
	ErrCaregiverNameRequired = errors.New("caregiver name is required")
	// ErrCaregiverTypeInvalid 代表未提供或提供了非既定選項的照護人員類型。
	ErrCaregiverTypeInvalid = errors.New("caregiver type must be case_manager or specialist")
)
