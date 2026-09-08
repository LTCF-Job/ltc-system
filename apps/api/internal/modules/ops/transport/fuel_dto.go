package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/ops/app"
)

// FuelLogResponse 是油資紀錄對外的 API 契約，欄位與
// apps/web/src/types/api.d.ts 的 FuelLogDTO 對齊。
type FuelLogResponse struct {
	ID          uuid.UUID  `json:"id"`
	VehicleID   uuid.UUID  `json:"vehicleId"`
	VehicleName string     `json:"vehicleName"`
	PlateNo     string     `json:"plateNo"`
	DriverID    *uuid.UUID `json:"driverId"`
	DriverName  *string    `json:"driverName"`
	FuelDate    string     `json:"fuelDate"`
	Liters      float64    `json:"liters"`
	Cost        float64    `json:"cost"`
	ReceiptURL  *string    `json:"receiptUrl"`
	CreatedBy   uuid.UUID  `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func newFuelLogResponse(f app.FuelLog) FuelLogResponse {
	return FuelLogResponse{
		ID:          f.ID,
		VehicleID:   f.VehicleID,
		VehicleName: f.VehicleName,
		PlateNo:     f.PlateNo,
		DriverID:    f.DriverID,
		DriverName:  f.DriverName,
		FuelDate:    formatDateOnly(f.FuelDate),
		Liters:      f.Liters,
		Cost:        f.Cost,
		ReceiptURL:  f.ReceiptURL,
		CreatedBy:   f.CreatedBy,
		CreatedAt:   f.CreatedAt,
	}
}

func newFuelLogResponses(list []app.FuelLog) []FuelLogResponse {
	out := make([]FuelLogResponse, 0, len(list))
	for _, f := range list {
		out = append(out, newFuelLogResponse(f))
	}
	return out
}
