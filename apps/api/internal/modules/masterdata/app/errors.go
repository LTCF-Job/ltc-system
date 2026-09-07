package app

import "errors"

var (
	// ErrDriverNotFound 代表查無司機資料。
	ErrDriverNotFound     = errors.New("driver not found")
	ErrDriverNameRequired = errors.New("driver name is required")
	// ErrNationalIDNotConfigured 代表資料尚未設定身分證密文。
	ErrNationalIDNotConfigured = errors.New("national id is not configured")
	// ErrRevealAuditUnavailable 代表高風險個資揭露無法留下稽核紀錄。
	ErrRevealAuditUnavailable = errors.New("reveal audit is unavailable")
	// ErrInvalidDriverNationalID 代表司機身分證檢查碼錯誤。
	ErrInvalidDriverNationalID = errors.New("invalid driver national id format")
	// ErrInvalidDriverLicenseClass 代表駕照類別不在允許的代碼清單內。
	ErrInvalidDriverLicenseClass = errors.New("invalid driver license class")
	ErrInvalidStatus             = errors.New("invalid status")
	ErrInvalidAssignmentRange    = errors.New("invalid driver assignment date range")
	// ErrRegionNameRequired 代表未提供區域名稱。
	ErrRegionNameRequired = errors.New("region name is required")
	// ErrDuplicateRegionName 代表區域名稱重複。
	ErrDuplicateRegionName = errors.New("region name already exists")
	// ErrRegionNotFound 代表查無區域資料。
	ErrRegionNotFound = errors.New("region not found")

	// ErrSiteNameRequired 代表未提供單位名稱。
	ErrSiteNameRequired = errors.New("site name is required")
	// ErrSiteAddressRequired 代表未提供單位地址。
	ErrSiteAddressRequired = errors.New("site address is required")
	// ErrSiteRegionRequired 代表未提供所屬區域。
	ErrSiteRegionRequired = errors.New("site region is required")
	// ErrDuplicateSiteName 代表該區域已存在相同名稱的單位。
	ErrDuplicateSiteName = errors.New("site name already exists in region")
	// ErrSiteNotFound 代表查無單位資料。
	ErrSiteNotFound = errors.New("site not found")

	// ErrDuplicateVehiclePlateNo 代表車號已存在。
	ErrDuplicateVehiclePlateNo = errors.New("vehicle plate number already exists")
	// ErrDuplicateVehicleDisplayName 代表車輛代稱已存在。
	ErrDuplicateVehicleDisplayName = errors.New("vehicle display name already exists")
	// ErrVehicleNotFound 代表查無車輛資料。
	ErrVehicleNotFound = errors.New("vehicle not found")
)
