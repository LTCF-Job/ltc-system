package govform

// GovClaimSheetName 政府申報表之標準工作表名稱（實測為「工作表1」）。
const GovClaimSheetName = "工作表1"

// Headers33 定義政府申報表第 1 列的 33 欄完整標題文字（含換行字元）。
var Headers33 = [33]string{
	"身分證字號",
	"服務日期(請輸入7碼)",
	"服務項目代碼",
	"服務類別\n1.補助\n2.自費",
	"數量\n(僅整數)",
	"單價",
	"服務人員身分證",
	"起始時段-小時\n(24小時制)",
	"起始時段-分鐘",
	"結束時段-小時\n(24小時制)",
	"結束時段-分鐘",
	"備註",
	"服務人員身分證2",
	"服務人員身分證3",
	"服務人員身分證4",
	"服務人員身分證5",
	"不申報AA09填1",
	"訪視未遇填1",
	"C碼必填-復能目標達成情形\n1.尚未滿服務組數\n2.滿服務組數且已達復能目標\n3.滿服務組數但尚未達復能目標\n4.未滿服務組數已結案，且已達復能目標\n5.未滿服務組數已結案，但未達復能目標",
	"C碼必填-復能目標",
	"C碼必填-指導對象",
	"C碼必填-服務內容",
	"C碼必填-指導建議摘要",
	"OT01必填-餐別\n1.早餐\n2.午餐\n3.晚餐",
	"BD03、DA01使用-出發地",
	"BD03、DA01使用-目的地",
	"BD03、DA01使用-出發地(緯度)",
	"BD03、DA01使用-出發地(經度)",
	"BD03、DA01使用-目的地(緯度)",
	"BD03、DA01使用-目的地(經度)",
	"BD03、DA01使用-里程數(公里)",
	"BD03、DA01使用-車號",
	"BD03必填-服務使用類型\n1.社區式長照機構\n2.社區服務據點(不含身障類)\n3.輔具中心\n4.身障日間照顧服務",
}

// RequiredHeaderCount 為政府申報表中以紅字標示的必填欄位數量。
// 官方範本將第 1～11 欄（身分證字號 ～ 結束時段-分鐘）標為紅字必填，
// 第 12 欄起為選填欄位並以黑字呈現。此區隔屬政府格式規範，
// 實際的字型與色碼由 reporting/infra 的 renderer 決定。
const RequiredHeaderCount = 11

// IsRequiredHeader 回報第 col 欄（1-based）是否為必填欄位。
func IsRequiredHeader(col int) bool {
	return col >= 1 && col <= RequiredHeaderCount
}

// DefaultServiceCode 為長照交通接送的服務項目代碼。
// 本系統的申報範圍僅有交通接送一種服務，故未指定時一律以此代碼申報。
const DefaultServiceCode = "BD03"

// DefaultUnitPrice 為長照交通接送的申報單價（新臺幣元）。
// case_schedules.unit_price 的 schema 預設值同為 115.00 且 NOT NULL，
// 個案尚未建立排班時以此值申報，避免必填欄位留白。
const DefaultUnitPrice = 115

// DirectionForLegSeq 由趟次序號推導去回程方向。
//
// 排班的趟次一律以「奇數去程、偶數回程」成對配置（二趟為 1 去 2 回，
// 四趟為 1 去 2 回 3 去 4 回），driverreport 的欄位對應也依此慣例。
// 個案尚未建立排班時 schedule_legs 沒有資料，但 ride_records.leg_seq
// 本身仍在，可據以還原方向。超出 1..4 範圍時回傳空字串表示無從判斷。
func DirectionForLegSeq(legSeq int16) string {
	if legSeq < 1 || legSeq > 4 {
		return ""
	}
	if legSeq%2 == 1 {
		return "outbound"
	}
	return "inbound"
}
