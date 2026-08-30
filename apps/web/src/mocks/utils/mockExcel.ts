import * as XLSX from 'xlsx'

/**
 * 建立標準合法之 Excel (.xlsx) 二進位 Blob，確保在 MSW Mock 開發環境下
 * 下載的所有 Excel 檔案（趟數表、時刻表、範本、申報表、保養表）在 Microsoft Excel 中皆能正常開啟不損毀。
 */

import * as XLSX from 'xlsx'

// 由 openpyxl 產生之合法 OpenXML 工作簿（個案匯入範本內容），內部 XML 屬性值皆正確加上雙引號
const VALID_DEMO_EXCEL_BASE64 =
  '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'

// 個案批次匯入範本專用工作簿：欄位名稱須與個案清單／匯入預覽表格一致（如「戶別」非「家戶類型」），據點與接送車輛皆填中文名稱，非系統編號
const CASE_IMPORT_TEMPLATE_EXCEL_BASE64 =
  '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' +
  'BQYAAAAACgAKAIACAADEFwAAAAA='

// 照護人員匯入範本專用工作簿：欄位順序為類型／單位／姓名／聯絡方式／備註，內容與後端
// RenderCaregiverImportTemplate 產生的範本一致（由該函式實際輸出後 base64 編碼而得）
const CAREGIVER_IMPORT_TEMPLATE_EXCEL_BASE64 =
  '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'

/**
 * 將 Base64 字串轉為標準 Blob 物件。
 */
export function base64ToBlob(
  base64: string,
  mimeType: string = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
): Blob {
  const byteCharacters = atob(base64)
  const bytes = new Uint8Array(byteCharacters.length)
  for (let i = 0; i < byteCharacters.length; i++) {
    bytes[i] = byteCharacters.charCodeAt(i)
  }
  return new Blob([bytes.buffer as ArrayBuffer], { type: mimeType })
}

/**
 * 產生合法的 Mock Excel 檔案 Blob。
 */
export function createMockExcelBlob(): Blob {
  return base64ToBlob(VALID_DEMO_EXCEL_BASE64)
}

/**
 * 產生個案批次匯入範本專用的 Excel Blob，欄位與範例值對齊個案基本資料表。
 */
export function createCaseImportTemplateExcelBlob(): Blob {
  return base64ToBlob(CASE_IMPORT_TEMPLATE_EXCEL_BASE64)
}

interface CaseProfileExportRow {
  name: string
  householdType?: string
  nationalId?: string
  gender?: string
  birthDate?: string
  siteName?: string
  outboundVehicle?: string
  inboundVehicle?: string
  careContactRole?: string
  careContactName?: string
  registeredAddress?: string
  homeAddress?: string
}

// 生日轉民國年 (YYY/MM/DD) 與依生日計算之歲數，欄位計算方式與後端 GenerateCaseProfileWorkbook 一致
function toRocBirthdayAndAge(birthDate?: string): [string, string] {
  if (!birthDate) return ['', '']
  const d = new Date(birthDate)
  if (Number.isNaN(d.getTime())) return ['', '']
  const rocYear = d.getFullYear() - 1911
  const birthday = `${String(rocYear).padStart(3, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}/${String(d.getDate()).padStart(2, '0')}`
  const age = String(new Date().getFullYear() - d.getFullYear())
  return [birthday, age]
}

/**
 * 產生「個案資料彙整」匯出用 Excel Blob，欄位與順序須與後端 RenderCaseProfileWorkbook 一致，
 * 依傳入的個案清單（可為使用者勾選後的子集合）逐列產生，而非固定範例資料。
 */
export function createCaseProfileExcelBlob(cases: CaseProfileExportRow[]): Blob {
  const headers = [
    '姓名', '戶別', '身分證字號', '性別', '生日', '歲數',
    '據點', '接送車輛(去)', '接送車輛(回)', '個管or照專', '姓名', '戶籍', '居住地', '備註'
  ]
  const rows = cases.map((item) => {
    const [birthday, age] = toRocBirthdayAndAge(item.birthDate)
    return [
      item.name, item.householdType || '', item.nationalId || '', item.gender || '',
      birthday, age, item.siteName || '', item.outboundVehicle || '', item.inboundVehicle || '',
      item.careContactRole || '', item.careContactName || '', item.registeredAddress || '',
      item.homeAddress || '', ''
    ]
  })
  const worksheet = XLSX.utils.aoa_to_sheet([headers, ...rows])
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, '進系統個案個資')
  const buffer = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer
  return new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
}

/**
 * 產生照護人員批次匯入範本專用的 Excel Blob，欄位與範例值對齊照護人員主檔（類型／單位／姓名／聯絡方式／備註）。
 */
export function createCaregiverImportTemplateExcelBlob(): Blob {
  return base64ToBlob(CAREGIVER_IMPORT_TEMPLATE_EXCEL_BASE64)
}

/**
 * 產生司機接送匯報範本 Blob。
 *
 * 此範本的欄位隨每台車已對應的個案而不同，無法像個案／照護人員範本那樣內嵌一份固定
 * base64，因此改為依後端 `RenderDriverReportTemplate` 的同一條組欄規則即時產生：
 * 民國日期、駕駛人、各個案趟次欄、備註，且只有表頭沒有示範資料列
 * （示範列會在匯入時被當成真實匯報寫入搭乘紀錄）。
 * 該規則由 apps/api 的 TestRenderDriverReportTemplate_RoundTrip 斷言。
 */
export function createDriverReportTemplateExcelBlob(caseColumns: string[]): Blob {
  const header = ['民國日期', '駕駛人', ...caseColumns, '備註']
  const sheet = XLSX.utils.aoa_to_sheet([header])
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, sheet, '司機接送匯報')
  const bytes = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' }) as ArrayBuffer
  return new Blob([bytes], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
  })
}
