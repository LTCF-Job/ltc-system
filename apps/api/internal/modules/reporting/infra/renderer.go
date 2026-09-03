package infra

// ExcelRenderer 以 excelize 產生報表檔案，是 reporting 唯一接觸試算表 SDK 的地方。
// 它直接接受 app 的報表型別：報表的欄位順序即檔案的欄位順序，再多一層等價的中繼
// 型別只會讓兩邊同步失敗時無聲產生錯欄。
type ExcelRenderer struct{}

// NewExcelRenderer 建立 ExcelRenderer 實例。
func NewExcelRenderer() ExcelRenderer { return ExcelRenderer{} }
