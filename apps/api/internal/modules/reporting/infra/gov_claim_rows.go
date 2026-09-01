package infra

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// govClaimSourceRow 對應 govClaimSourceQuery 的一列結果。
type govClaimSourceRow struct {
	CaseID               uuid.UUID
	CaseCode             string
	CaseName             string
	Region               string
	CaseNationalIDCipher []byte
	CaseNationalIDMasked string
	HomeAddress          string
	ServiceCategory      int
	ServiceUsageType     int

	ServiceDate    time.Time
	LegSeq         int16
	NotClaimedAA09 bool
	Direction      *string
	DepartTime     *string
	DurationMin    *int

	ServiceCode string
	UnitPrice   float64
	DistanceKM  float64
	SiteAddress string
	PlateNo     string

	DriverID               *uuid.UUID
	DriverNationalIDCipher []byte
}

func (r govClaimSourceRow) toApp() app.GovClaimSource {
	return app.GovClaimSource{
		CaseID:               r.CaseID,
		CaseCode:             r.CaseCode,
		CaseName:             r.CaseName,
		Region:               r.Region,
		CaseNationalIDCipher: r.CaseNationalIDCipher,
		CaseNationalIDMasked: r.CaseNationalIDMasked,
		HomeAddress:          r.HomeAddress,
		ServiceCategory:      r.ServiceCategory,
		ServiceUsageType:     r.ServiceUsageType,

		ServiceDate:    r.ServiceDate,
		LegSeq:         r.LegSeq,
		NotClaimedAA09: r.NotClaimedAA09,
		Direction:      r.Direction,
		DepartTime:     r.DepartTime,
		DurationMin:    r.DurationMin,

		ServiceCode: r.ServiceCode,
		UnitPrice:   r.UnitPrice,
		DistanceKM:  r.DistanceKM,
		SiteAddress: r.SiteAddress,
		PlateNo:     r.PlateNo,

		DriverID:               r.DriverID,
		DriverNationalIDCipher: r.DriverNationalIDCipher,
	}
}
