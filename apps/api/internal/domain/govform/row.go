package govform

import (
	"fmt"
	"strings"
	"time"

	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/domain/timeslot"
)

// ClaimRowInput 代表產生單列 33 欄資料所需的輸入來源。
type ClaimRowInput struct {
	NationalIDPlain  string
	ServiceDate      time.Time
	ServiceCode      string
	ServiceCategory  int     // 1 或 2
	UnitPrice        float64 // 115.00
	DriverNationalID string
	DepartTime       time.Time
	DurationMin      int
	NotClaimedAA09   bool
	Direction        string // "outbound" 或 "inbound"
	LegSeq           int16
	HomeAddress      string
	SiteAddress      string
	DistanceKM       float64
	PlateNo          string
	ServiceUsageType int // 1..4
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
// 來源缺漏或不合法的欄位一律留白後照樣產出該列：政府申報採部分匯出，使用者要先拿到
// 報得出來的資料，缺什麼由匯出結果的資料缺漏清單提示，而不是整列或整批被擋下來。
func BuildClaimRow(input ClaimRowInput) (ClaimRow, error) {
	var row ClaimRow
	row.ServiceDate = input.ServiceDate
	row.Direction = input.Direction
	row.LegSeq = input.LegSeq
	row.NationalID = input.NationalIDPlain

	// 1. 身分證字號 (文字)
	row.Cells[0] = strings.TrimSpace(input.NationalIDPlain)

	// 2. 服務日期 (數值，民國 7 碼，例：1150701)
	row.Cells[1] = ""
	if !input.ServiceDate.IsZero() {
		rocDateInt, err := rocdate.ToROC(input.ServiceDate)
		if err != nil {
			return row, fmt.Errorf("failed to convert date to ROC: %w", err)
		}
		row.Cells[1] = rocDateInt
	}

	// 3. 服務項目代碼 (文字)
	// 本系統只申報長照交通接送，代碼固定為 BD03（case_schedules.service_code
	// 的 schema 預設值亦為 BD03 且 NOT NULL）。個案尚未建立排班時 ServiceCode
	// 會是空字串，此處補上預設值，避免申報檔出現空白的必填欄位。
	row.Cells[2] = strings.TrimSpace(input.ServiceCode)
	if row.Cells[2] == "" {
		row.Cells[2] = DefaultServiceCode
	}

	// 4. 服務類別 (數值: 1 補助 / 2 自費)
	row.Cells[3] = ""
	if input.ServiceCategory == 1 || input.ServiceCategory == 2 {
		row.Cells[3] = input.ServiceCategory
	}

	// 5. 數量 (數值: 固定 1)
	row.Cells[4] = 1

	// 6. 單價 (數值)
	row.Cells[5] = ""
	if input.UnitPrice > 0 {
		if input.UnitPrice == float64(int(input.UnitPrice)) {
			row.Cells[5] = int(input.UnitPrice)
		} else {
			row.Cells[5] = input.UnitPrice
		}
	}

	// 7. 服務人員身分證 (文字)
	row.Cells[6] = strings.TrimSpace(input.DriverNationalID)

	// 8-11. 起始與結束時段 (數值，不補零；結束時段經 timeslot 跨小時進位運算)。
	// 出發時間必須與服務日期同一天才有意義，對不上時四欄一起留白。
	row.Cells[7] = ""
	row.Cells[8] = ""
	row.Cells[9] = ""
	row.Cells[10] = ""
	sameDay := !input.DepartTime.IsZero() && !input.ServiceDate.IsZero() &&
		input.DepartTime.Year() == input.ServiceDate.Year() &&
		input.DepartTime.YearDay() == input.ServiceDate.YearDay()
	if sameDay {
		row.Cells[7] = input.DepartTime.Hour()
		row.Cells[8] = input.DepartTime.Minute()
		if input.DurationMin > 0 {
			endHour, endMin, err := timeslot.EndTime(input.DepartTime, input.DurationMin)
			if err != nil {
				return row, fmt.Errorf("failed to calculate end time: %w", err)
			}
			row.Cells[9] = endHour
			row.Cells[10] = endMin
		}
	}

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

	// 25, 26. 出發地與目的地 (R1 去回程地址對調)。
	// 方向不明時無從判斷哪一邊是出發地，兩欄一起留白，不猜一個方向填進去。
	row.Cells[24] = ""
	row.Cells[25] = ""
	switch input.Direction {
	case "inbound":
		row.Cells[24] = strings.TrimSpace(input.SiteAddress)
		row.Cells[25] = strings.TrimSpace(input.HomeAddress)
	case "outbound":
		row.Cells[24] = strings.TrimSpace(input.HomeAddress)
		row.Cells[25] = strings.TrimSpace(input.SiteAddress)
	}

	// 27-30. 經緯度 (空字串)
	row.Cells[26] = ""
	row.Cells[27] = ""
	row.Cells[28] = ""
	row.Cells[29] = ""

	// 31. 里程數 (數值)
	row.Cells[30] = ""
	if input.DistanceKM > 0 {
		if input.DistanceKM == float64(int(input.DistanceKM)) {
			row.Cells[30] = int(input.DistanceKM)
		} else {
			row.Cells[30] = input.DistanceKM
		}
	}

	// 32. 車號 (文字)
	row.Cells[31] = strings.TrimSpace(input.PlateNo)

	// 33. 服務使用類型 (數值: 1..4)
	row.Cells[32] = ""
	if input.ServiceUsageType >= 1 && input.ServiceUsageType <= 4 {
		row.Cells[32] = input.ServiceUsageType
	}

	return row, nil
}
