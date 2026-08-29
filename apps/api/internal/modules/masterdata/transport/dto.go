package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// 本檔的 response DTO 是 masterdata 對外的 API 契約。json tag 與搬遷前
// repository entity 逐欄一致，搬遷不得改變任何既有回應形狀；轉換函式對 nil
// slice 回傳 nil，維持清單為空時序列化成 null 的既有行為。

// SiteResponse 代表回傳給前端的據點資料。
type SiteResponse struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Region    string    `json:"region"`
	OpenDays  []int16   `json:"openDays"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newSiteResponse(s app.Site) SiteResponse {
	return SiteResponse{
		ID:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		Address:   s.Address,
		Region:    s.Region,
		OpenDays:  s.OpenDays,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func newSiteResponses(list []app.Site) []SiteResponse {
	if list == nil {
		return nil
	}
	out := make([]SiteResponse, 0, len(list))
	for _, s := range list {
		out = append(out, newSiteResponse(s))
	}
	return out
}

// CreateSiteRequest 代表新增據點請求。既有 API 未對任何欄位強制必填，維持不變。
type CreateSiteRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Region   string  `json:"region"`
	OpenDays []int16 `json:"openDays"`
	Status   string  `json:"status"`
}

// UpdateSiteRequest 代表更新據點請求。
type UpdateSiteRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Region   string  `json:"region"`
	OpenDays []int16 `json:"openDays"`
	Status   string  `json:"status"`
}

// VehicleResponse 代表回傳給前端的車輛資料。
type VehicleResponse struct {
	ID          uuid.UUID `json:"id"`
	PlateNo     string    `json:"plateNo"`
	DisplayName string    `json:"displayName"`
	Region      string    `json:"region"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newVehicleResponse(v app.Vehicle) VehicleResponse {
	return VehicleResponse{
		ID:          v.ID,
		PlateNo:     v.PlateNo,
		DisplayName: v.DisplayName,
		Region:      v.Region,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func newVehicleResponses(list []app.Vehicle) []VehicleResponse {
	if list == nil {
		return nil
	}
	out := make([]VehicleResponse, 0, len(list))
	for _, v := range list {
		out = append(out, newVehicleResponse(v))
	}
	return out
}

// CreateVehicleRequest 代表新增車輛請求。既有 API 未強制任何必填欄位，維持不變。
type CreateVehicleRequest struct {
	PlateNo     string `json:"plateNo"`
	DisplayName string `json:"displayName"`
	Region      string `json:"region"`
	Status      string `json:"status"`
}

// UpdateVehicleRequest 代表更新車輛請求。
type UpdateVehicleRequest struct {
	PlateNo     string `json:"plateNo"`
	DisplayName string `json:"displayName"`
	Region      string `json:"region"`
	Status      string `json:"status"`
}

// DriverResponse 代表回傳給前端的司機資料。身分證密文與 HMAC 索引不對外輸出。
type DriverResponse struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	NameNormalized   string    `json:"nameNormalized"`
	NationalIDMasked string    `json:"nationalIdMasked"`
	Email            *string   `json:"email,omitempty"`
	Region           string    `json:"region"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func newDriverResponse(d app.Driver) DriverResponse {
	return DriverResponse{
		ID:               d.ID,
		Code:             d.Code,
		Name:             d.Name,
		NameNormalized:   d.NameNormalized,
		NationalIDMasked: d.NationalIDMasked,
		Email:            d.Email,
		Region:           d.Region,
		Status:           d.Status,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func newDriverResponses(list []app.Driver) []DriverResponse {
	if list == nil {
		return nil
	}
	out := make([]DriverResponse, 0, len(list))
	for _, d := range list {
		out = append(out, newDriverResponse(d))
	}
	return out
}

// DriverAssignmentResponse 代表司機車輛指派結果。
type DriverAssignmentResponse struct {
	ID             uuid.UUID  `json:"id"`
	DriverID       uuid.UUID  `json:"driverId"`
	DriverName     string     `json:"driverName,omitempty"`
	VehicleID      uuid.UUID  `json:"vehicleId"`
	VehicleName    string     `json:"vehicleName,omitempty"`
	VehiclePlateNo string     `json:"vehiclePlateNo,omitempty"`
	IsPrimary      bool       `json:"isPrimary"`
	EffectiveFrom  time.Time  `json:"effectiveFrom"`
	EffectiveTo    *time.Time `json:"effectiveTo,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func newDriverAssignmentResponse(a app.DriverAssignment) DriverAssignmentResponse {
	return DriverAssignmentResponse{
		ID:             a.ID,
		DriverID:       a.DriverID,
		DriverName:     a.DriverName,
		VehicleID:      a.VehicleID,
		VehicleName:    a.VehicleName,
		VehiclePlateNo: a.VehiclePlateNo,
		IsPrimary:      a.IsPrimary,
		EffectiveFrom:  a.EffectiveFrom,
		EffectiveTo:    a.EffectiveTo,
		CreatedAt:      a.CreatedAt,
	}
}

// CreateDriverRequest 代表新增司機請求。
type CreateDriverRequest struct {
	Code       string  `json:"code"`
	Name       string  `json:"name" binding:"required"`
	NationalID string  `json:"nationalId" binding:"required"`
	Email      *string `json:"email"`
	Region     string  `json:"region" binding:"required"`
}

// UpdateDriverRequest 代表更新司機請求，欄位為 nil 表示不變更。
type UpdateDriverRequest struct {
	Name   *string `json:"name"`
	Email  *string `json:"email"`
	Region *string `json:"region"`
	Status *string `json:"status"`
}

// AssignVehicleRequest 代表指派司機車輛請求。
type AssignVehicleRequest struct {
	VehicleID     uuid.UUID  `json:"vehicleId" binding:"required"`
	IsPrimary     bool       `json:"isPrimary"`
	EffectiveFrom time.Time  `json:"effectiveFrom" binding:"required"`
	EffectiveTo   *time.Time `json:"effectiveTo"`
}

// RegionResponse 代表回傳給前端的區域資料。
type RegionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newRegionResponse(r app.Region) RegionResponse {
	return RegionResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		SortOrder:   r.SortOrder,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func newRegionResponses(list []app.Region) []RegionResponse {
	if list == nil {
		return nil
	}
	out := make([]RegionResponse, 0, len(list))
	for _, r := range list {
		out = append(out, newRegionResponse(r))
	}
	return out
}

// CreateRegionRequest 代表新增區域請求。
type CreateRegionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateRegionRequest 代表更新區域請求，欄位為 nil 表示不變更。
type UpdateRegionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	SortOrder   *int    `json:"sortOrder"`
}
