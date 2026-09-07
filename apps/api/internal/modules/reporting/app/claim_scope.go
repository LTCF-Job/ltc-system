package app

import (
	"time"

	"github.com/google/uuid"
)

// ClaimScope 是申報前置檢核與申報來源查詢共用的資料範圍。
type ClaimScope struct {
	StartDate time.Time
	EndDate   time.Time
	Region    *string
	CaseIDs   []uuid.UUID
}

// NewClaimScope 建立申報範圍；空白 region 代表不限制區域。
func NewClaimScope(startDate, endDate time.Time, region string, caseIDs []uuid.UUID) ClaimScope {
	var regionPtr *string
	if region != "" {
		regionCopy := region
		regionPtr = &regionCopy
	}
	return ClaimScope{
		StartDate: startDate,
		EndDate:   endDate,
		Region:    regionPtr,
		CaseIDs:   append([]uuid.UUID(nil), caseIDs...),
	}
}

// RegionValue 取得 SQL 查詢使用的區域值；nil 代表不限制區域。
func (s ClaimScope) RegionValue() string {
	if s.Region == nil {
		return ""
	}
	return *s.Region
}
