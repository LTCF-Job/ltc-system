package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	// 設定驗證器使用結構體的 json tag 作為欄位名稱，避免回傳 Go 內部 struct 欄位名
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

// 常用欄位中文名稱對照表，提供易讀的中文錯誤描述
var commonFieldLabels = map[string]string{
	"name":                      "名稱",
	"siteName":                  "單位名稱",
	"address":                   "地址",
	"region":                    "區域",
	"openDays":                  "開放星期",
	"status":                    "狀態",
	"plateNo":                   "車牌號碼",
	"displayName":               "顯示名稱",
	"siteId":                    "所屬單位",
	"nationalId":                "身分證字號",
	"email":                     "電子信箱",
	"code":                      "代碼",
	"brand":                     "廠牌",
	"model":                     "型號",
	"manufactureYm":             "出廠年月",
	"compulsoryInsuranceExpiry": "強制險到期日",
	"passengerInsuranceExpiry":  "乘客險到期日",
	"thirdPartyInsuranceExpiry": "第三人責任險到期日",
	"lastInspectionDate":        "最後驗車日",
	"wheelchairAccessible":      "輪椅友善",
}

// ExtractValidationDetails 從 binding 與 validation 錯誤中提煉出友善的欄位詳細資訊。
func ExtractValidationDetails(err error) []ErrorDetail {
	if err == nil {
		return nil
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		details := make([]ErrorDetail, 0, len(valErrs))
		for _, fe := range valErrs {
			fieldName := fe.Field()
			label := commonFieldLabels[fieldName]
			if label == "" {
				label = fieldName
			}

			var reason string
			switch fe.Tag() {
			case "required":
				reason = fmt.Sprintf("%s為必填項目", label)
			case "min":
				reason = fmt.Sprintf("%s長度或數值不得小於 %s", label, fe.Param())
			case "max":
				reason = fmt.Sprintf("%s長度或數值不得大於 %s", label, fe.Param())
			case "email":
				reason = fmt.Sprintf("%s格式不正確", label)
			case "uuid":
				reason = fmt.Sprintf("%s必須為有效 UUID", label)
			case "oneof":
				reason = fmt.Sprintf("%s必須為下列選項之一：%s", label, fe.Param())
			default:
				reason = fmt.Sprintf("%s不符合驗證規則 (%s)", label, fe.Tag())
			}

			details = append(details, ErrorDetail{
				Field:  fieldName,
				Reason: reason,
			})
		}
		return details
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		fieldName := typeErr.Field
		label := commonFieldLabels[fieldName]
		if label == "" {
			label = fieldName
		}
		return []ErrorDetail{
			{
				Field:  fieldName,
				Reason: fmt.Sprintf("%s資料型態錯誤，預期為 %s", label, typeErr.Type.String()),
			},
		}
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return []ErrorDetail{
			{
				Reason: "JSON 格式錯誤",
			},
		}
	}

	if errors.Is(err, io.EOF) {
		return []ErrorDetail{
			{
				Reason: "請求內容不能為空",
			},
		}
	}

	return []ErrorDetail{
		{
			Reason: "輸入資料不符合格式要求",
		},
	}
}
