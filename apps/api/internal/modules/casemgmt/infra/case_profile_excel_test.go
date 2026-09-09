package infra

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
)

// 表頭與資料位置必須逐格與來源工作表「進系統個案個資」一致，因此以回讀產出的
// 工作簿驗證，而不是只確認 render 沒有回傳錯誤。
func TestRenderCaseProfileWorkbook_MatchesSourceLayout(t *testing.T) {
	data, err := ExcelRenderer{}.RenderCaseProfileWorkbook([]app.CaseProfileRow{{
		Name:              "王小明",
		HouseholdType:     "一般",
		NationalID:        "A202559750",
		Gender:            "男",
		Birthday:          "045/06/15",
		SiteName:          "竹南日照",
		CareContactRole:   "個管",
		CareContactName:   "陳小華",
		RegisteredAddress: "苗栗縣竹南鎮戶籍地址",
		HomeAddress:       "苗栗縣竹南鎮居住地址",
		Remarks:           "需輪椅",
	}})
	require.NoError(t, err)
	require.NotEmpty(t, data)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	rows, err := f.GetRows("進系統個案個資")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, []string{
		"姓名", "戶別", "身分證字號", "性別", "生日", "據點",
		"個管or照專", "照護人員", "戶籍", "居住地", "備註",
	}, rows[0])

	assert.Equal(t, []string{
		"王小明", "一般", "A202559750", "男", "045/06/15", "竹南日照",
		"個管", "陳小華", "苗栗縣竹南鎮戶籍地址", "苗栗縣竹南鎮居住地址", "需輪椅",
	}, rows[1])
}
