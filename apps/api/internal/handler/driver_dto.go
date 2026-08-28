package handler

import (
	"time"

	"github.com/google/uuid"
)

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

// CreateVehicleRequest 代表新增車輛請求，欄位需求與既有 repository.VehicleEntity binding 行為一致（無強制必填）。
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
