package govform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovClaim_TsaiTsengChieh_OutputContract(t *testing.T) {
	// 模擬蔡曾切 115 年 7 月之資料生成
	// 個案身分證: A202559750, 服務代碼: BD03, 服務類別: 1, 單價: 115, 時長: 10 分鐘
	// 司機身分證: K120098177, 車號: BZG-7915, 里程: 5, 使用類型: 2
	// 戶籍地址: 新竹縣竹北市光明六路264號, 單位地址: 新竹縣竹北市中正西路100號

	var rows []ClaimRow

	// 去程: 7/1 ~ 7/29 (共 19 筆，09:40 -> 09:50)
	// 回程: 7/1 ~ 7/30 (共 20 筆，16:00 -> 16:10)
	for day := 1; day <= 19; day++ {
		date := time.Date(2026, 7, day, 0, 0, 0, 0, time.UTC)
		dep := time.Date(2026, 7, day, 9, 40, 0, 0, time.UTC)
		r, err := BuildClaimRow(ClaimRowInput{
			NationalIDPlain:  "A202559750",
			ServiceDate:      date,
			ServiceCode:      "BD03",
			ServiceCategory:  1,
			UnitPrice:        115.0,
			DriverNationalID: "K120098177",
			DepartTime:       dep,
			DurationMin:      10,
			Direction:        "outbound",
			LegSeq:           1,
			HomeAddress:      "新竹縣竹北市光明六路264號",
			SiteAddress:      "新竹縣竹北市中正西路100號",
			DistanceKM:       5.0,
			PlateNo:          "BZG-7915",
			ServiceUsageType: 2,
		})
		require.NoError(t, err)
		rows = append(rows, r)
	}

	for day := 1; day <= 20; day++ {
		date := time.Date(2026, 7, day, 0, 0, 0, 0, time.UTC)
		dep := time.Date(2026, 7, day, 16, 0, 0, 0, time.UTC)
		r, err := BuildClaimRow(ClaimRowInput{
			NationalIDPlain:  "A202559750",
			ServiceDate:      date,
			ServiceCode:      "BD03",
			ServiceCategory:  1,
			UnitPrice:        115.0,
			DriverNationalID: "K120098177",
			DepartTime:       dep,
			DurationMin:      10,
			Direction:        "inbound",
			LegSeq:           2,
			HomeAddress:      "新竹縣竹北市光明六路264號",
			SiteAddress:      "新竹縣竹北市中正西路100號",
			DistanceKM:       5.0,
			PlateNo:          "BZG-7915",
			ServiceUsageType: 2,
		})
		require.NoError(t, err)
		rows = append(rows, r)
	}

	SortClaimRows(rows, false)
	assert.Len(t, rows, 39)

	// 驗證去回程交界點出發地與目的地之反轉 (R1)
	// 第 0 列為去程 -> 出發地為住家
	assert.Equal(t, "新竹縣竹北市光明六路264號", rows[0].Cells[24])
	assert.Equal(t, "新竹縣竹北市中正西路100號", rows[0].Cells[25])

	// 第 19 列為回程 -> 出發地為單位
	assert.Equal(t, "新竹縣竹北市中正西路100號", rows[19].Cells[24])
	assert.Equal(t, "新竹縣竹北市光明六路264號", rows[19].Cells[25])

	// 驗證第 1 筆之各項儲存格型別與數值（AC-3）
	assert.Equal(t, "A202559750", rows[0].Cells[0]) // 身分證: 文字
	assert.Equal(t, 1150701, rows[0].Cells[1])      // 日期: 數值
	assert.Equal(t, "BD03", rows[0].Cells[2])       // 項目: 文字
	assert.Equal(t, 1, rows[0].Cells[3])            // 類別: 數值
	assert.Equal(t, 1, rows[0].Cells[4])            // 數量: 數值
	assert.Equal(t, 115, rows[0].Cells[5])          // 單價: 數值
	assert.Equal(t, "K120098177", rows[0].Cells[6]) // 服務人員: 文字
	assert.Equal(t, 9, rows[0].Cells[7])            // 起始時: 數值
	assert.Equal(t, 40, rows[0].Cells[8])           // 起始分: 數值
	assert.Equal(t, 9, rows[0].Cells[9])            // 結束時: 數值
	assert.Equal(t, 50, rows[0].Cells[10])          // 結束分: 數值
	assert.Equal(t, 5, rows[0].Cells[30])           // 里程: 數值
	assert.Equal(t, "BZG-7915", rows[0].Cells[31])  // 車號: 文字
	assert.Equal(t, 2, rows[0].Cells[32])           // 服務使用類型: 數值
}

// TestGovClaim_BlankColumnsMatchGovernmentSample 鎖定範本 蔡曾切11507.xls 中留白的 20 個欄位。
// 這些欄位目前系統沒有資料來源，任何一欄意外被填值都會讓政府端收到無法解讀的申報資料。
func TestGovClaim_BlankColumnsMatchGovernmentSample(t *testing.T) {
	row, err := BuildClaimRow(ClaimRowInput{
		NationalIDPlain:  "A202559750",
		ServiceDate:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		ServiceCode:      "BD03",
		ServiceCategory:  1,
		UnitPrice:        115.0,
		DriverNationalID: "K120098177",
		DepartTime:       time.Date(2026, 7, 1, 9, 40, 0, 0, time.UTC),
		DurationMin:      10,
		Direction:        "outbound",
		LegSeq:           1,
		HomeAddress:      "新竹縣竹北市光明六路264號",
		SiteAddress:      "新竹縣竹北市中正西路100號",
		DistanceKM:       5.0,
		PlateNo:          "BZG-7915",
		ServiceUsageType: 2,
	})
	require.NoError(t, err)

	blankIndexes := []int{
		11,             // 備註
		12, 13, 14, 15, // 服務人員身分證 2-5
		16,                 // 不申報AA09（本列未標記）
		17,                 // 訪視未遇
		18, 19, 20, 21, 22, // C 碼五欄
		23,             // OT01 餐別
		26, 27, 28, 29, // 出發地／目的地經緯度
	}
	for _, idx := range blankIndexes {
		assert.Equal(t, "", row.Cells[idx], "第 %d 欄應留白（Headers33[%d]=%q）", idx+1, idx, Headers33[idx])
	}
}

// TestGovClaim_NotClaimedAA09 標記不申報時第 17 欄才寫入數值 1。
func TestGovClaim_NotClaimedAA09(t *testing.T) {
	input := ClaimRowInput{
		NationalIDPlain:  "A202559750",
		ServiceDate:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		ServiceCode:      "BD03",
		ServiceCategory:  1,
		UnitPrice:        115.0,
		DriverNationalID: "K120098177",
		DepartTime:       time.Date(2026, 7, 1, 9, 40, 0, 0, time.UTC),
		DurationMin:      10,
		Direction:        "outbound",
		LegSeq:           1,
		DistanceKM:       5.0,
		PlateNo:          "BZG-7915",
		ServiceUsageType: 2,
		NotClaimedAA09:   true,
	}
	row, err := BuildClaimRow(input)
	require.NoError(t, err)
	assert.Equal(t, 1, row.Cells[16])
}
