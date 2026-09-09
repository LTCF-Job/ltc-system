package app

// ImportService 負責批次 Excel 個案資料之解析、預覽與匯入。
type ImportService struct {
	cases           CaseRegistrar
	duplicates      CaseDuplicateFinder
	duplicateStager DuplicateCandidateStager
	siteRepo        SiteLookup
	vehicleRepo     VehicleLookup
	caregiverRepo   CaregiverLookup
	prefRepo        TransportPreferenceWriter
	spreadsheet     SpreadsheetReader
	template        TemplateRenderer
	txRunner        TxRunner
	idempotency     CaseImportIdempotencyStore
}

// NewImportService 建立 ImportService 實例。
func NewImportService(
	cases CaseRegistrar,
	duplicates CaseDuplicateFinder,
	duplicateStager DuplicateCandidateStager,
	siteRepo SiteLookup,
	vehicleRepo VehicleLookup,
	caregiverRepo CaregiverLookup,
	prefRepo TransportPreferenceWriter,
	spreadsheet SpreadsheetReader,
	template TemplateRenderer,
	txRunner TxRunner,
) *ImportService {
	return &ImportService{
		cases:           cases,
		duplicates:      duplicates,
		duplicateStager: duplicateStager,
		siteRepo:        siteRepo,
		vehicleRepo:     vehicleRepo,
		caregiverRepo:   caregiverRepo,
		prefRepo:        prefRepo,
		spreadsheet:     spreadsheet,
		template:        template,
		txRunner:        txRunner,
	}
}

// SetIdempotencyStore 設定正式資料庫使用的匯入列冪等儲存；離線測試可省略。
func (s *ImportService) SetIdempotencyStore(store CaseImportIdempotencyStore) {
	s.idempotency = store
}

// CaseImportTemplateExcel 產生批次匯入的 Excel 範本位元組。
func (s *ImportService) CaseImportTemplateExcel() ([]byte, error) {
	return s.template.RenderCaseImportTemplate()
}
