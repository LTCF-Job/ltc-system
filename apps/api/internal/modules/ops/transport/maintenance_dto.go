package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/ops/app"
)

// MaintenanceLogResponse 是車輛維修保養紀錄對外的 API 契約，欄位與
// apps/web/src/types/api.d.ts 的 MaintenanceLogDTO 對齊。
type MaintenanceLogResponse struct {
	ID          uuid.UUID `json:"id"`
	VehicleID   uuid.UUID `json:"vehicleId"`
	VehicleName string    `json:"vehicleName"`
	PlateNo     string    `json:"plateNo"`
	ServiceDate string    `json:"serviceDate"`
	Mileage     float64   `json:"mileage"`
	Items       string    `json:"items"`
	Vendor      *string   `json:"vendor"`
	Cost        float64   `json:"cost"`
	ReceiptURL  *string   `json:"receiptUrl"`
	Note        *string   `json:"note"`
	CreatedBy   uuid.UUID `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

func newMaintenanceLogResponse(m app.MaintenanceLog) MaintenanceLogResponse {
	return MaintenanceLogResponse{
		ID:          m.ID,
		VehicleID:   m.VehicleID,
		VehicleName: m.VehicleName,
		PlateNo:     m.PlateNo,
		ServiceDate: formatDateOnly(m.ServiceDate),
		Mileage:     m.Mileage,
		Items:       m.Items,
		Vendor:      m.Vendor,
		Cost:        m.Cost,
		ReceiptURL:  m.ReceiptURL,
		Note:        m.Note,
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt,
	}
}

func newMaintenanceLogResponses(list []app.MaintenanceLog) []MaintenanceLogResponse {
	out := make([]MaintenanceLogResponse, 0, len(list))
	for _, m := range list {
		out = append(out, newMaintenanceLogResponse(m))
	}
	return out
}
