package infra

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// exportJobRow 對應 export_jobs 的一列查詢結果。
// TotalCases 與 TotalRows 只在列表查詢時由聚合子查詢填入，單筆查詢改由檔案清單推算。
type exportJobRow struct {
	ID           uuid.UUID
	JobType      string
	PeriodYM     string
	Region       string
	Format       string
	Status       string
	ErrorMessage *string
	CreatedAt    time.Time
	FinishedAt   *time.Time
	TotalCases   int
	TotalRows    int
}

func (r exportJobRow) toApp() app.GovClaimJob {
	mode := app.GovClaimModeDirect
	if r.Format == "zip" {
		mode = app.GovClaimModeZip
	}
	errorMessage := ""
	if r.ErrorMessage != nil {
		errorMessage = *r.ErrorMessage
	}
	return app.GovClaimJob{
		ID:           r.ID,
		JobType:      r.JobType,
		PeriodYM:     r.PeriodYM,
		Region:       r.Region,
		Mode:         mode,
		Status:       r.Status,
		TotalCases:   r.TotalCases,
		TotalRows:    r.TotalRows,
		ErrorMessage: errorMessage,
		CreatedAt:    r.CreatedAt,
		FinishedAt:   r.FinishedAt,
	}
}

// exportLineRow 對應 export_lines 的一列查詢結果。
type exportLineRow struct {
	LineNo           int
	CaseID           uuid.UUID
	NationalIDMasked string
	ServiceDateROC   int
	RawPayload       []byte
}

func (r exportLineRow) toApp() (app.ExportLine, error) {
	var payload app.ClaimLinePayload
	if err := json.Unmarshal(r.RawPayload, &payload); err != nil {
		return app.ExportLine{}, fmt.Errorf("unmarshal export line payload (line %d): %w", r.LineNo, err)
	}
	return app.ExportLine{
		LineNo:           r.LineNo,
		CaseID:           r.CaseID,
		NationalIDMasked: r.NationalIDMasked,
		ServiceDateROC:   r.ServiceDateROC,
		Payload:          payload,
	}, nil
}
