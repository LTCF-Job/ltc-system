package app

import (
	"time"

	"github.com/google/uuid"
)

// ClaimScope 是申報前置檢核與申報來源查詢共用的資料範圍。
type ClaimScope struct {
	StartDate time.Time
	EndDate   time.Time
	CaseIDs   []uuid.UUID
}

// NewClaimScope 建立申報範圍。
func NewClaimScope(startDate, endDate time.Time, caseIDs []uuid.UUID) ClaimScope {
	return ClaimScope{
		StartDate: startDate,
		EndDate:   endDate,
		CaseIDs:   append([]uuid.UUID(nil), caseIDs...),
	}
}
