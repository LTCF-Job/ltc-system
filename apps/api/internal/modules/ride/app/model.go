package app

import (
	"time"

	"github.com/google/uuid"
)

// FormColumn 代表 form_columns 實體。
type FormColumn struct {
	ID              uuid.UUID
	FormID          uuid.UUID
	ColumnIndex     int
	ColumnHeader    string
	CleanedName     string
	Kind            string
	MappingStatus   string
	CaseID          *uuid.UUID
	LegSeq          *int16
	SuggestedCaseID *uuid.UUID
	SuggestionScore float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RideRecord 代表 ride_records 實體。
type RideRecord struct {
	ID                     uuid.UUID
	CaseID                 uuid.UUID
	CaseName               string
	ServiceDate            time.Time
	LegSeq                 int16
	MergedStatus           string
	EffectiveStatus        string
	VehicleID              uuid.UUID
	VehicleName            string
	DriverID               *uuid.UUID
	DriverName             string
	HasConflict            bool
	ConflictResolvedAt     *time.Time
	ConflictResolvedBy     *uuid.UUID
	ConflictResolutionNote *string
	DepartTimeOverride     *string
	DurationMinOverride    *int16
	NotClaimedAA09         bool
	CorrectedBy            *uuid.UUID
	CorrectedAt            *time.Time
	CorrectionReason       *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
