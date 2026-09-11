package app

import "errors"

var (
	// ErrDriverNotFound 代表查無司機資料。
	ErrDriverNotFound     = errors.New("driver not found")
	ErrDriverNameRequired = errors.New("driver name is required")
	// ErrInvalidDriverNationalID 代表司機身分證檢查碼錯誤。
	ErrInvalidDriverNationalID = errors.New("invalid driver national id format")
	// ErrInvalidDriverLicenseClass 代表駕照類別不在允許的代碼清單內。
	ErrInvalidDriverLicenseClass = errors.New("invalid driver license class")
	// ErrInvalidDriverEmail 代表司機電子信箱格式不正確。
	ErrInvalidDriverEmail = errors.New("invalid driver email format")
	// ErrDuplicateNationalID 代表身分證字號已被其他司機登記。
	ErrDuplicateNationalID    = errors.New("driver national id already exists")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrInvalidAssignmentRange = errors.New("invalid driver assignment date range")

	// ErrSiteNameRequired 代表未提供據點名稱。
	ErrSiteNameRequired = errors.New("site name is required")
	// ErrSiteAddressRequired 代表未提供據點地址。
	ErrSiteAddressRequired = errors.New("site address is required")
	// ErrDuplicateSiteName 代表該區域已存在相同名稱的據點。
	ErrDuplicateSiteName = errors.New("site name already exists in region")
	// ErrSiteNotFound 代表查無據點資料。
	ErrSiteNotFound = errors.New("site not found")
	// ErrSiteInUse 代表刪除據點時仍有其他資料（如個案排班）參照該據點，資料庫外鍵限制擋下。
	ErrSiteInUse = errors.New("site is still referenced by other records")

	// ErrAssignmentReferenceInvalid 代表指派時指定的司機或車輛不存在（外鍵違反）。
	ErrAssignmentReferenceInvalid = errors.New("assigned driver or vehicle does not exist")
	// ErrAssignmentOverlap 代表司機的指派期間與既有指派重疊（違反不重疊限制）。
	ErrAssignmentOverlap = errors.New("driver assignment overlaps with an existing assignment")

	// ErrDuplicateVehiclePlateNo 代表車號已存在。
	ErrDuplicateVehiclePlateNo = errors.New("vehicle plate number already exists")
	// ErrDuplicateVehicleDisplayName 代表車輛車別已存在。
	ErrDuplicateVehicleDisplayName = errors.New("vehicle display name already exists")
	// ErrVehicleNotFound 代表查無車輛資料。
	ErrVehicleNotFound = errors.New("vehicle not found")
)
