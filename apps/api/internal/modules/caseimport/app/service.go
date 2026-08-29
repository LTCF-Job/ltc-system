package app

// ImportService 負責批次 Excel／CSV 個案資料之解析、預覽與匯入。
type ImportService struct {
	cases       CaseRegistrar
	siteRepo    SiteLookup
	vehicleRepo VehicleLookup
	prefRepo    TransportPreferenceWriter
	spreadsheet SpreadsheetReader
	template    TemplateRenderer
	txRunner    TxRunner
}

// NewImportService 建立 ImportService 實例。
func NewImportService(
	cases CaseRegistrar,
	siteRepo SiteLookup,
	vehicleRepo VehicleLookup,
	prefRepo TransportPreferenceWriter,
	spreadsheet SpreadsheetReader,
	template TemplateRenderer,
	txRunner TxRunner,
) *ImportService {
	return &ImportService{
		cases:       cases,
		siteRepo:    siteRepo,
		vehicleRepo: vehicleRepo,
		prefRepo:    prefRepo,
		spreadsheet: spreadsheet,
		template:    template,
		txRunner:    txRunner,
	}
}

// CaseImportTemplateExcel 產生批次匯入的 Excel 範本位元組。
func (s *ImportService) CaseImportTemplateExcel() ([]byte, error) {
	return s.template.RenderCaseImportTemplate()
}
