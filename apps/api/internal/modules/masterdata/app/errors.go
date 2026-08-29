package app

import "errors"

var (
	// ErrDriverNotFound 代表查無司機資料。
	ErrDriverNotFound = errors.New("driver not found")
	// ErrInvalidDriverNationalID 代表司機身分證檢查碼錯誤。
	ErrInvalidDriverNationalID = errors.New("invalid driver national id format")
	// ErrRegionNameRequired 代表未提供區域名稱。
	ErrRegionNameRequired = errors.New("region name is required")
	// ErrDuplicateRegionName 代表區域名稱重複。
	ErrDuplicateRegionName = errors.New("region name already exists")
	// ErrRegionNotFound 代表查無區域資料。
	ErrRegionNotFound = errors.New("region not found")
)
