---
doc_type: decision
covers:
  - apps/api/migrations/000044_cases_site_link.up.sql
  - apps/api/migrations/000045_drop_schedule_and_preference_site.up.sql
  - apps/api/migrations/000046_vehicles_site_text.up.sql
  - apps/api/migrations/000047_caregivers_site_text.up.sql
  - apps/api/internal/modules/casemgmt/
  - apps/api/internal/modules/masterdata/
  - apps/api/internal/modules/caregiver/
  - apps/api/internal/modules/caseimport/infra/excel.go
  - apps/api/internal/modules/caregiver/infra/excel.go
---

# 據點（單位）關聯重構：從排班／偏好搬到個案本身，車輛與照護人員解耦

決策日期：2026-09-09

## Context

「單位」主檔（`sites` 表）原本被四個地方各自引用一份 FK：`case_schedules.site_id`、
`case_transport_preferences.site_id`、`vehicles.site_id`、`caregivers.site_id`。同一個個案的
排班與交通偏好理論上該指向同一個據點，卻各自存一份，容易對不上；車輛與照護人員的「所屬單位」
在實務上只是備註用途，卻背著完整的主檔比對與待維護機制，車輛還因此背了一個唯讀 `region`（由
所屬單位 JOIN 帶出）。

同時，UI 與文件用語從「單位」全面改為「據點」（`sites` 表本身、Go 識別碼 `SiteID`／`siteId`、
權限鍵 `masters_sites` 不變，只改中文顯示）。

## Decision

1. **據點關聯改成個案自己持有**：新增 `cases.site_id`（nullable FK，`ON DELETE RESTRICT`）與
   `cases.site_name_raw`，並以 `CHECK ck_cases_site_present` 要求兩者至少一個非空。排班
   （`case_schedules`）與交通偏好（`case_transport_preferences`）上原本重複的據點欄位移除
   （migration `000045`），兩者一律改讀個案本身的 `site_id`。應搭日曆的 `open_days` 來源、政府
   申報匯出與新竹接送時刻表的據點地址／名稱，全部從「排班／交通偏好的據點」改成「個案的據點」；
   交集比對本身的規則不變，只是 SQL JOIN 的起點換了。
2. **`cases.site_id` 資料庫可為 NULL，API 層才要求必填**：手動新增個案 `POST /cases` 的
   `siteId` 是 `binding:"required"`，但資料庫欄位本身允許 NULL——這是刻意的落差，讓 Excel
   匯入路徑可以在據點名稱比對不到主檔時退回 `site_name_raw`，靠 `CHECK` 約束（至少一個非空）
   而非 `NOT NULL` 把關，觸發既有的待維護機制（`site_pending`，`site_id IS NULL AND
   site_name_raw IS NOT NULL`）而不是直接擋下整筆匯入。
3. **車輛與照護人員的據點徹底解耦成自由文字**：`vehicles.site_id`／`caregivers.site_id`
   （與 `caregivers.site_name_raw`）皆移除，改為單純的 `site_name TEXT`，不驗證、不比對
   `sites` 主檔、非必填（migration `000046`／`000047`）。車輛原本由所屬單位 JOIN 出來的唯讀
   `region` 欄位一併移除，連帶拿掉 `GET /vehicles?region=`、趟次月結表區域篩選、新竹接送時刻表
   「只顯示新竹車輛」篩選；司機接送匯報原本的區域 fallback 改為寫死常數 `'hsinchu'`。
4. **Excel 匯入表頭只認「據點」**：個案與照護人員匯入範本（`caseimport/infra/excel.go`、
   `caregiver/infra/excel.go`）的表頭改成只接受「據點」，原本相容的「單位」表頭拿掉，範本檔案
   同步重新產生。

## Alternatives

- **保留車輛／照護人員的 `site_id` FK，只是改成選填**：沒選，因為這兩處的「所屬單位」從未真正
  參與過需要主檔完整性的業務邏輯（不像個案的據點會決定應搭日曆與申報地址），保留 FK 只是徒增
  「比對不到主檔就變成待維護」這種不必要的維護負擔，且車輛的 `region` 唯讀欄位本身也沒有其他
  用途在依賴它。
- **`cases.site_id` 直接設 `NOT NULL`**：沒選，因為既有資料與 Excel 匯入無法保證匯入當下據點
  名稱一定能在主檔裡找到對應列，`NOT NULL` 會讓匯入必須整批擋下而非讓單筆落入待維護——與準則三
  ／準則四的「非阻擋、待維護」設計精神衝突。
- **Excel 匯入表頭同時接受「單位」與「據點」以維持回溯相容**：使用者明確決定不保留，避免表頭
  長期存在兩套同義詞造成匯入範本與教學文件的維護負擔。

## Consequences

- **不可逆**：`case_schedules.site_id` 原本允許「同一個案不同排班掛不同據點」的資料模型，
  drop 後這種歷史差異無法還原；同理，車輛與照護人員原本的據點 FK 關聯也回不去，往後只能重新
  手動輸入。
- 待維護判定新增 `cases.site_pending`，與既有 `profile_pending`、`link_pending`（現在只剩
  去／回程車輛兩個條件）一起被 `case_pending_status` view 用 `OR` 合併，詳見
  [pending-data-visibility.md](pending-data-visibility.md) 與
  `docs/tech/system-logic-specification.md` 準則四。
- 舊版 Excel 檔案若表頭仍是「單位」，匯入時該欄不會被辨識，需要使用者重新下載範本或手動把表頭
  改成「據點」。
- 車輛與照護人員的 `siteName` 純粹是自由文字備註，畫面上不再提供「連結到既有據點」的操作
  （`PUT /caregivers/:id/site` 與對應的「新增單位快速建立」對話框已移除）。
