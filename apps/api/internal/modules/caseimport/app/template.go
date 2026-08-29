package app

import "strings"

// GenerateCaseImportTemplateCSV 產生個案批次匯入標準 CSV 範本文字，對齊「進系統個案個資」欄位格式。
func GenerateCaseImportTemplateCSV() string {
	var sb strings.Builder
	sb.WriteString(string(rune(0xFEFF)))
	sb.WriteString("姓名*,戶別,身分證字號,性別,生日,據點,接送車輛(去),接送車輛(回),個管or照專,姓名(個管/照專),戶籍,居住地,REMARK" + crlf)
	sb.WriteString("張曾阿妹,一般戶,A202559750,女,1945/06/15,竹南日照據點,竹南1車,竹南2車,個管,陳小華,苗栗縣竹南鎮戶籍地址,苗栗縣竹南鎮大營路123號,行動不便需輪椅" + crlf)
	sb.WriteString("李國盛,低收入戶,G121806465,男,1950/02/20,竹北日照中心,竹北1車,竹北2車,照專,王小明,新竹縣竹北市戶籍地址,新竹縣竹北市文興路一段200號," + crlf)
	return sb.String()
}

const crlf = "\r\n"
