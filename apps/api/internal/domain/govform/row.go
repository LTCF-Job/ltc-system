package govform

import (
	"fmt"
	"time"

	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/domain/timeslot"
)

// ClaimRowInput 代表產生單列 33 欄資料所需的輸入來源。
type ClaimRowInput struct {
	NationalIDPlain     string
	ServiceDate         time.Time
	ServiceCode         string
	ServiceCategory     int     // 1 或 2
	UnitPrice           float64 // 115.00
	DriverNationalID    string
	DepartTime          time.Time
	DurationMin         int
	NotClaimedAA09      bool
	Direction           string // "outbound" 或 "inbound"
	LegSeq              int16
	HomeAddress         string
	SiteAddress         string
	DistanceKM          float64
	PlateNo             string
	ServiceUsageType    int // 1..4
}

// ClaimRow 代表單行 33 欄各儲存格的精確型別與數值（符合規格書 7.3）。
type ClaimRow struct {
	// 33 欄各格值（儲存 interface{}，數值為 int/float64，字串為 string）
	Cells [33]interface{}

	// 用於排序的中繼資料
	ServiceDate time.Time
	Direction   string
	LegSeq      int16
	NationalID  string
}

// BuildClaimRow 依據規格書 7.3 與 7.4 將業務實體轉換為精確型別的 33 欄資料。
func BuildClaimRow(input ClaimRowInput) (ClaimRow, error) {
	var row ClaimRow
	row.ServiceDate = input.ServiceDate
	row.Direction = input.Direction
	row.LegSeq = input.LegSeq
	row.NationalID = input.NationalIDPlain

	// 1. 身分證字號 (文字)
	row.Cells[0] = input.NationalIDPlain

	// 2. 服務日期 (數值，民國 7 碼，例：1150701)
	rocDateInt, err := rocdate.ToROC(input.ServiceDate)
	if err != nil {
		return row, fmt.Errorf("failed to convert date to ROC: %w", err)
	}
	row.Cells[1] = rocDateInt

	// 3. 服務項目代碼 (文字)
	row.Cells[2] = input.ServiceCode

	// 4. 服務類別 (數值: 1 補助 / 2 自費)
	row.Cells[3] = input.ServiceCategory

	// 5. 數量 (數值: 固定 1)
	row.Cells[4] = 1

	// 6. 單價 (數值)
	if input.UnitPrice == float64(int(input.UnitPrice)) {
		row.Cells[5] = int(input.UnitPrice)
	} else {
		row.Cells[5] = input.UnitPrice
	}

	// 7. 服務人員身分證 (文字)
	row.Cells[6] = input.DriverNationalID

	// 8, 9. 起始時段 (數值，不補零)
	depHour := input.DepartTime.Hour()
	depMin := input.DepartTime.Minute()
	row.Cells[7] = depHour
	row.Cells[8] = depMin

	// 10, 11. 結束時段 (數值，經 timeslot 跨小時進位運算)
	duration := input.DurationMin
	if duration <= 0 {
		duration = 10
	}
	endHour, endMin, err := timeslot.EndTime(input.DepartTime, duration)
	if err != nil {
		return row, fmt.Errorf("failed to calculate end time: %w", err)
	}
	row.Cells[9] = endHour
	row.Cells[10] = endMin

	// 12. 備註 (空字串)
	row.Cells[11] = ""

	// 13-16. 服務人員身分證 2-5 (空字串)
	row.Cells[12] = ""
	row.Cells[13] = ""
	row.Cells[14] = ""
	row.Cells[15] = ""

	// 17. 不申報AA09 (數值 1 或空字串)
	if input.NotClaimedAA09 {
		row.Cells[16] = 1
	} else {
		row.Cells[16] = ""
	}

	// 18. 訪視未遇 (空字串)
	row.Cells[17] = ""

	// 19-23. C碼欄位 (空字串)
	row.Cells[18] = ""
	row.Cells[19] = ""
	row.Cells[20] = ""
	row.Cells[21] = ""
	row.Cells[22] = ""

	// 24. OT01餐別 (空字串)
	row.Cells[23] = ""

	// 25, 26. 出發地與目的地 (R1 去回程地址對調)
	if input.Direction == "inbound" {
		row.Cells[24] = input.SiteAddress
		row.Cells[25] = input.HomeAddress
	} else {
		row.Cells[24] = input.HomeAddress
		row.Cells[25] = input.SiteAddress
	}

	// 27-30. 經緯度 (空字串)
	row.Cells[26] = ""
	row.Cells[27] = ""
	row.Cells[28] = ""
	row.Cells[29] = ""

	// 31. 里程數 (數值)
	if input.DistanceKM == float64(int(input.DistanceKM)) {
		row.Cells[30] = int(input.DistanceKM)
	} else {
		row.Cells[30] = input.DistanceKM
	}

	// 32. 車號 (文字)
	row.Cells[31] = input.PlateNo

	// 33. 服務使用類型 (數值: 1..4)
	row.Cells[32] = input.ServiceUsageType

	return row, nil
}
