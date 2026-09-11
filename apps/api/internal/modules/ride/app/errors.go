package app

import "errors"

// ErrRideNotFound 表示查無指定的搭乘紀錄。
var ErrRideNotFound = errors.New("ride record not found")

// ErrConflictAlreadyResolved 表示該筆混車衝突已被裁決過。
var ErrConflictAlreadyResolved = errors.New("conflict already resolved")

// ErrInvalidManualRideLeg 表示人工補登的個案／日期沒有對應的有效排班趟次。
var ErrInvalidManualRideLeg = errors.New("manual ride leg is not scheduled")

// ErrManualRideVehicleRequired 表示有效排班趟次與請求都沒有可用車輛。
var ErrManualRideVehicleRequired = errors.New("manual ride vehicle is required")

// ErrInvalidRideCorrectionField 表示 PATCH 嘗試清除資料庫不可為 NULL 的欄位。
var ErrInvalidRideCorrectionField = errors.New("invalid ride correction field")

// ErrInvalidManualReportStatus 表示人工補登請求的搭乘狀態不是 boarded／absent。
// 錯誤訊息本身即為使用者可讀的中文原因，handler 端可直接沿用。
var ErrInvalidManualReportStatus = errors.New("無效的搭乘狀態")

// ErrInvalidManualReportServiceDate 表示人工補登請求的服務日期格式錯誤。
var ErrInvalidManualReportServiceDate = errors.New("無效的服務日期格式")
