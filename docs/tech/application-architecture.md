---
doc_type: architecture
covers:
  - apps/api/
  - apps/web/
---

# 應用程式目標架構與開發規範

日期：2026-09-10。狀態：**目標設計，尚未實作遷移**。

本文件供新增功能、重構與 code review 遵循：後端採 **Clean Architecture＋DDD 的模組化單體**；前端採 **Vue 3 功能模組架構（feature-oriented architecture）**。目標依業務需求制定，不以既有目錄限制設計；遷移則沿用可用的程式碼與公開契約。

本文件負責架構方向、責任歸屬及開發規則，不取代 [系統業務規範](system-logic-specification.md)、[API 規格](api-design-specification.md)、[整合契約](integration-contract.md) 與各業務決策。既有 [後端框架](backend-framework.md)、[前端框架](frontend-framework.md) 描述目前實作。本文末尾僅定義遷移階段，不是已排定的實作任務清單。

## 1. 架構決策

| 面向 | 決策 | 理由與成本 |
|---|---|---|
| 後端部署 | 單一 Go API、模組化單體 | 現有操作常跨資料寫入與稽核，需要同一資料庫交易；目前沒有必須獨立部署的需求 |
| 後端內部 | 每個 bounded context 內分 domain / application / adapters | 業務規則可獨立測試；代價是邊界需要明確型別轉換 |
| DDD 深度 | 核心流程採聚合與 value object，簡單主檔採薄領域模型 | 不為每張資料表建立 domain service、factory 或事件 |
| 資料存取 | 保留 pgx 與 SQL，repository 實作放外層 | Clean Architecture 不要求換 ORM 或資料庫 |
| 前端 | Vue 3、TypeScript、Vite、Vue Router、Element Plus | 適合目前以登入後 CRUD、表格、匯入及報表為主的 SPA；沒有 SSR／SEO 驅動的框架更換需求 |
| 前端組織 | app / pages / features / shared | 讓同一功能的 UI、資料存取與互動邏輯集中；不採完整 FSD 多層規格 |
| 狀態 | TanStack Vue Query 管伺服器資料；Pinia 管跨頁用戶端狀態 | 快取與失效責任明確；需要逐功能導入 Query，不能與原 store 重複持有同份資料 |
| 系統整合 | HTTP 契約隔離前後端 | 前端不複製 Go 聚合、不直接讀業務資料庫、不自行決定正式業務合法性 |

TanStack Vue Query 是**預計新增**的依賴，目前 `apps/web/package.json` 尚未安裝。實作時選擇與當時 Vue／TypeScript 相容的穩定版本並更新 lockfile；本文不指定升級既有套件。

不納入本次目標：微服務、Event Sourcing、訊息匯流排、分離讀寫資料庫、通用 CRUD framework、全站 UI 重畫或一次性重寫。DDD 不等於上述技術；先把業務語意、依賴與一致性界線做對。

## 2. 目前實作與目標的差距

以下是 develop 工作樹的靜態觀察；路徑相對 repo 根目錄，未執行測試或 runtime 驗證。

| 現況證據（路徑／symbol） | 影響與目標調整 | 後續驗證 |
|---|---|---|
| `apps/api/internal/arch/arch_test.go`：`allowedInternal`（L54-57 對 `to.kind()=="domain"` 無條件放行）、`zoneOf`（L357 未知第一段回 `""`）、`internalRoot=".."`（L20 不掃描 `cmd/server`）、`externalConfinement`（L76-87 綁死 transport/infra 字面 kind）、`isPersistencePackage`（L335 以 `HasSuffix(name,"infra")` 判斷）；目前為 transport/app/infra，baseline 四張皆空 | baseline 空白是「過渡期違規已清空」，不是「新 layout 已受保護」：五個破口目前預設 fail-open（module-local domain 可被任何模組 import、`internal/sharedkernel`／`internal/bootstrap` 不受掃描、新 adapters 分段會被第三方圍堵表擋下 gin／pgx）。新增 domain、application、adapters 前必須先改寫 zone 模型，把未知區域改成 fail-closed | 架構測試加入合法／非法 import 案例，並針對五個破口各補一個會失敗的反例 |
| `apps/api/internal/modules/ride/app/ride_service.go`：`RideService`、`sourceFingerprint`、`CorrectRideRecord` | 更正流程、快照與模型集中在 app；將不變條件移進 ride/domain，使用案例保留協調 | 更正、來源變更及人工保護的特徵測試 |
| `apps/api/internal/domain/merge`、`calendar`、`govform` | 全域 domain 混合多種業務所有權；依下表搬回擁有者，非單純改目錄名稱 | 呼叫端映射及演算法結果不變 |
| `apps/api/cmd/server/module_adapters.go`：`rideDriverResolver`、`rideScheduleReader` 直接持有其他模組 infra repository | 跨模組可能繞過提供端 application 政策；改呼叫提供端公開查詢／使用案例。`rideDriverResolver.GetByNameNormalized` 已把 `masterapp.ErrDriverNotFound` 轉成 `(nil, nil)`，屬業務判斷洩漏到 composition root，遷移時要隨 bridge 一起移除 | 停用、刪除、待維護篩選與提供端契約測試 |
| `apps/api/internal/modules/masterdata/app/site_service.go`：`Create`、`Update` 呼叫 `writeAuditBestEffort`（`audit.go:22`，失敗只 `slog.Error`，不回傳 error） | 依 §6「稽核等級判準」與 `mutation-audit-policy.md`，一般主檔 create／update 屬非阻斷性，此實作是照既有決策寫的，**不是待修正項**；只有 `delete` 屬阻斷性，才需要同交易回滾 | 依判準分別驗證：非阻斷路徑注入 audit failure 應仍寫入業務列並記 log；阻斷路徑注入失敗應回滾 |
| `apps/api/internal/platform/pgxdb/txrunner.go`：`TxRunner.WithTx`（L37 無 nested 偵測，無條件 `pool.Begin`）、`NewTxRunner(nil)` 回傳 `nil`（L29-34）、`FromContext` 的 fallback 設計（L62） | 交易 context 機制可用，但目前不具備巢狀合併：外層已有交易時 `WithTx` 仍會再開一條新交易；`nil` runner 與 `FromContext` fallback 會讓寫入在缺交易 context 時靜默走 pool 自動提交。現在未爆出雙交易是因為 `ride/app` 尚無 txRunner（巧合，非設計）。目標行為可參考 `casemgmt/infra/case_repo.go:365`（`CreateSchedule` 以 `TxFromContext` 判斷併入外層） | 多 repository 同交易、外層已有交易時內層不得再開新交易、nil-runner 缺交易寫入應回傳錯誤而非自動提交 |
| `apps/web/src/router/index.ts`、`views/masters/SiteListView.vue`、`api/masters.ts` | 目前為 views/api 等技術分類，沒有 features 目錄；按功能切片聚合。`SiteListView.vue` 僅 384 行，是主檔頁面中最單純的一支，本切片不驗證大型畫面的拆分方式 | 路由與 API 契約不變、相關元件／狀態驗證 |
| `apps/web/src/api/client.ts` 頂層 import auth store、router、Element Plus | HTTP 層耦合登入與呈現；改為注入 token provider 與認證失效 callback，錯誤提示移給使用案例呈現層 | 401 單次登出、錯誤不重複提示、Blob 錯誤解析 |
| 全後端 `grep "AND updated_at" apps/api/internal/modules/ --include=*.go` 無命中；唯一 409 是 `identity/transport/errors.go:22` 的 `RESOURCE_IN_USE`，與併發無關 | 樂觀鎖／expected version 全系統尚未實作；§5 更正流程圖與 §11 階段二、三「版本衝突可驗證」目前沒有任何一處可驗證的基礎，需要獨立的契約前置工作，見 §11 階段零 | SQL 版本條件、DTO 攜帶 expected version、409 對應的前端重新載入流程 |
| `apps/web`：無 `.eslintrc*`／`eslint.config.*`／`.dependency-cruiser*`，`package.json` 無 `lint` script，repo 根目錄非 npm workspace | 前端「建立可執行 import 規則」是從零建立整套工具鏈（新 devDependency＋script＋CI），與後端擴充既有 `arch_test.go` 不是同一量級，排程時不可視為同等工作量 | shared 不依賴上層、features 不互讀內部檔案的規則各補一個會失敗的反例 |

這是遷移定位，並非全系統缺陷清單。稽核等級的判準見 §6「稽核等級判準」，與 `mutation-audit-policy.md`、`mutation-audit-specification.md` §3.1 一致；不可只搬檔案便宣稱符合新標準。

## 3. 後端 bounded context 與共同語言

Bounded context 是一組一致的業務語意與模型邊界，不等於單一資料表或部署服務。以下是本專案採用的初始邊界；實作某切片前，須以其資料寫入、呼叫者及不變條件覆核。

| Context／目標模組 | 所有權與主要模型 | 現有來源 |
|---|---|---|
| 個案管理 `casemgmt` | Case、Schedule；個案資料、排班規則、交通偏好、個案待維護、個案匯入 | casemgmt＋caseimport；calendar 核心 |
| 運輸資源 `masterdata` | Site、Driver、Vehicle、Assignment；各自為獨立聚合 | masterdata |
| 照護人員 `caregiver` | Caregiver 生命週期與其匯入 | caregiver |
| 接送匯報 `driverreport` | ReportForm、ColumnMapping、Submission；欄位對應、原始匯報、匯入作業 | driverreport |
| 搭乘執行 `ride` | RideRecord、RideSource、Correction；實際搭乘狀態、合併、裁決與更正 | ride＋merge |
| 營運紀錄 `ops` | Attendance、FuelLog、MaintenanceRecord | ops |
| 報表申報 `reporting` | 查詢投影、前置檢核、申報資料列；不是大型可寫聚合 | reporting＋govform |
| 行事曆 `holiday` | Holiday、行事曆同步 | holiday |
| 身分權限 `identity` | 使用者安全狀態、Role、Permission；外部身分服務透過 adapter | identity |
| 稽核 `audit` | AuditEntry、稽核查詢與唯一 SQL 寫入所有權 | audit |
| 通知 `notification` | 收件設定、寄送紀錄、傳送 port | notification |

`task` 是工作排程入口，目標移到 `internal/bootstrap/jobs`，呼叫各 context 的 application；缺報的判定仍歸 ride，申報規則歸 reporting。不為排程另建一份領域模型。caseimport 是個案匯入的使用案例與檔案 adapter，目標併回 casemgmt；不建立全站共用 ImportAggregate。

共同語言：Schedule 表示「應搭乘」，RideRecord 表示「實際搭乘」，Submission 表示「原始回報」，ColumnMapping 表示「欄位對應」，彼此不可用同一 struct 代表。pending、active/inactive、deleted_at 是不同維度；是否顯示必須依 [待維護隔離決策](../decisions/pending-data-visibility.md) 與系統規範判斷。

```text
masterdata / caregiver / holiday
             | 公開查詢
             v
         casemgmt ------> driverreport
             |                | 提交已解析回報
             v                v
             +-------------> ride ------> reporting

identity 提供 actor／權限；audit 接收同交易稽核；notification 處理外部通知。
箭頭表示資料／服務流向，不表示 Go package 可直接跨模組 import。
```

車輛的據點文字與照護人員的單位文字仍為自由文字；不可因模型整理而改成 Site 外鍵。個案的 Site 關聯與待維護規則沿用現行業務規格。

## 4. 後端目錄與依賴規則

以下 `domain`／`application`／`adapters` 是目標命名。現行程式碼是 `transport`／`app`／`infra`（見 [後端框架](backend-framework.md)），`internal/arch/arch_test.go` 目前只認得這三段。某模組完成切片並且 `arch_test.go` 已擴充到能辨識其新 layout 前，該模組維持現行命名仍然有效；不得只照本節命名寫程式碼而讓 `go test ./...` 失敗。

```text
apps/api/
  cmd/server/main.go                 啟動入口
  internal/bootstrap/               組裝、路由、跨模組 bridge、jobs
  internal/modules/ride/
    domain/                         聚合、value object、領域政策、領域錯誤
      ride.go
      correction.go
      merge_policy.go
    application/                    使用案例、command/query、port、結果模型
      correct_ride.go
      list_rides.go
      ports.go
    adapters/
      http/                         Gin handler、HTTP DTO、error mapping
      postgres/                     repository、row mapping、SQL
  internal/platform/                DB、HTTP、設定、logging 等技術實作
  internal/sharedkernel/            極少量穩定且跨 context 的純值／規則
  internal/arch/                    可執行的依賴檢查
```

只有需要時才建立檔案與 adapter 目錄；簡單主檔不必每個 use case 一檔。固定 Go package 名稱採 `domain`、`application`、`http`、`postgres`；composition root 用 `rideapp`、`ridehttp` 等 alias 消除歧義。

| 來源 | 允許依賴 | 禁止 |
|---|---|---|
| domain | 標準庫、核准的純值型別、sharedkernel | application、adapters、platform、其他 context、HTTP／DB／SDK |
| application | 自己的 domain、自己定義的 port、sharedkernel | adapters、platform 技術實作、其他 context package |
| adapters | 自己的 application/domain、platform、外部 SDK | 其他 context 內部 package、在 handler 寫 SQL |
| sharedkernel | 標準庫與明確核准的純值依賴 | modules、platform |
| platform | 技術依賴、必要 sharedkernel | modules 與 bootstrap |
| bootstrap | 上述所有層 | 業務判斷與 SQL |

`application -> domain` 是編譯依賴；application 呼叫自己定義的 repository port，由 bootstrap 注入 postgres adapter，**執行時呼叫外層不代表內層 import 外層**。

跨 context 採消費端 port：例如 ride/application 宣告 ScheduleReader 與自己需要的 ScheduleSnapshot；bootstrap bridge 呼叫 casemgmt/application 公開 query 並轉型。不得傳 casemgmt 聚合給 ride，不得直接呼叫其 postgres repository。bridge 只轉型，不持有規則或交易決策。

既有 `internal/domain` 分配：merge → ride/domain（`driverreport/app/{driver_report_service.go,parse.go}` 目前直接 import 它，遷移時需改經 port，否則會變成跨 context import）；calendar → casemgmt/domain；govform 的申報規則 → reporting/domain，Excel 欄位版型／繪製 → reporting/adapters/spreadsheet。timeslot 唯一 importer 是 `internal/domain/govform/row.go:105`，modules 與 cmd 皆無直接使用，與 govform 一起歸 reporting/domain，不進 sharedkernel。rocdate、namenorm 盤點後橫跨多個 module（rocdate 7 個以上、namenorm 5 個），維持 sharedkernel。crypto 中身分證檢查碼驗證是跨 context 的純值規則，歸 sharedkernel——實際 importer 橫跨 caseimport、casemgmt、masterdata（司機身分證）與 reporting 四個模組，放進 casemgmt/domain 會讓 masterdata、reporting 產生跨 context import，違反本節的依賴矩陣；加解密、金鑰與 HMAC 實作仍放外層，由 application port 隔離。

## 5. DDD 實作規範

### 聚合與 value object

- **Entity** 有穩定身分，例如 Case、RideRecord；不要以姓名識別實體。
- **Value object** 以值判等並在建立時驗證，例如 ServiceDate、LegSeq、RideStatus；不為一般描述文字強制包一層型別。
- **Aggregate root** 提供狀態變更方法並守住不變條件；外界不得直接修改內部集合或以 setter 繞過驗證。讀取回傳 snapshot，避免可變 slice 洩漏。
- **Domain service／policy** 僅容納不適合單一 entity 的純規則，例如多來源合併；不得查 DB、寫 log、讀系統時間或發通知。需要的時間由參數提供。
- domain 不帶 `json`、`binding` 或 DB tag。HTTP DTO、application command/result、domain、persistence row 在責任不同時明確轉換。

初始聚合設計：Case 持有個案自身狀態；Schedule 是有獨立版本的聚合，以 CaseID 參照個案，不把所有歷史排班塞進 Case。RideRecord 以個案／服務日／趟次辨識同一業務紀錄，掌管合併結果與人工覆寫保護；來源歷程透過 repository 查詢，避免每次載入無界集合。ReportForm 持有有限的欄位對應設定；大量 Submission 另存，不成為表單聚合的巨大 children。

這些是目標模型，不是新 schema 的立即要求。正式採用前需確認既有唯一鍵與來源 fingerprint 足以表達一致性；若需新增 version／unique constraint，獨立規劃 migration。

### application 與 port

application 按動作命名，例如 `CorrectRide`、`CommitDriverReportImport`、`UpdateCase`；負責載入、授權、交易範圍、呼叫 domain、儲存、稽核及回傳結果。避免把所有動作堆進一個萬用 Service。

port 統一由消費端 application 擁有，包括 aggregate repository、read query、transaction runner、clock、ID generator、外部查詢與通知。即使目前只有一個 DB 實作，隔離外部 IO 仍需要 port；不因只有一個實作而讓 application import pgx。domain 保持純粹，不直接呼叫 repository。

repository 以聚合與使用情境定義方法；禁止 `GenericRepository[T]` 與逐資料表 DAO 洩漏到 use case。查詢可用 query port 直接回傳 application read model，不必載入聚合或重跑狀態變更規則。這是同資料庫內的讀寫責任分離，不另建 CQRS 系統。

### 更正搭乘紀錄的標準流程

```text
HTTP DTO -> CorrectRide command（actor、expected version、來源 fingerprint）
  -> application 檢查操作權限
  -> 開啟交易，載入目前紀錄與來源，完成鎖定／版本檢查
  -> domain.Correct(...) 驗證狀態、趟次與人工更正規則
  -> repository.Save(..., expectedVersion)
  -> AuditWriter.Append(明確且不含敏感資料的 snapshot)
  -> commit -> result -> HTTP DTO
```

domain 驗證無法取代資料庫 concurrency control。來源 fingerprint 比對與 Save 必須有同一鎖定／版本策略，避免「檢查完又被別人改掉」。PATCH command 保留未提供／null／值三態；不可把未提供誤當清空。

## 6. 交易、失敗與外部副作用

application 決定需要原子完成的範圍，postgres adapter 執行 begin/commit/rollback。交易 port 可維持 `WithTx(context.Context, func(context.Context) error) error`；pgx 型別與 context key 只存在於外層。

- 同次操作的 repository、跨 context application 與稽核必須收到同一交易 context，且使用同一 pool／DB。bootstrap bridge 不開新交易。
- 目標 TxRunner 遇到已存在的交易須加入外層交易；僅最外層提交。內層錯誤不得被吞掉後讓外層提交。需要 savepoint 時另設明確 port，不以巢狀 WithTx 假裝 savepoint。現行 `pgxdb.WithTx`（`txrunner.go:37`）沒有這層偵測，會無條件 `pool.Begin` 開新交易；目前沒有爆出雙交易，是因為 `ride/app` 尚無 txRunner，屬巧合而非設計。`casemgmt/infra/case_repo.go:365`（`CreateSchedule`）以 `TxFromContext` 判斷是否併入外層，可作為目標行為的參考實作。
- 必須在交易內執行的寫入若缺少交易 context，回傳錯誤，不得回退 pool 自動提交。一般查詢才可使用無交易連線。現行 `NewTxRunner(nil)` 回傳 `nil` 而非 error（`txrunner.go:29`），`ops/app/attendance_service.go:416` 的 `runInTx` 在 `txRunner == nil` 時直接 `fn(ctx)` 無交易執行，兩者都是待收斂的降級路徑。
- 新增／遷移的本地 mutation 以「業務資料＋必要稽核同交易」為目標，但「必要」依下方稽核等級判準決定，不是全面同交易。讀取觀測可 best effort；外部操作依專用流程處理。既有 best-effort mutation 依判準若已屬非阻斷級，不視為待修正項；若判準要求阻斷而現況仍是 best-effort，才在遷移表中標為尚未符合。
- 版本衝突回傳穩定 application error，transport 映射 409；不存在與衝突須分辨，不能只憑 UPDATE 零列便一律報衝突。**現況**：全系統沒有任何一處實作樂觀鎖（`grep "AND updated_at" apps/api/internal/modules/` 無命中），這條是目標行為，實作範圍見 §11 階段零。
- DB unique constraint 是併發下的最終防線；先查唯一性可改善訊息，但不能取代 constraint。

### 稽核等級判準

「必要稽核」不是模糊詞，判準句：**若稽核寫入失敗而業務資料保留，是否會讓該操作事後無法被查核或追責？** 是 → 阻斷性；否 → 非阻斷性。與 [`mutation-audit-policy.md`](../decisions/mutation-audit-policy.md) 的分級一致，該文件為權威來源，本節只是判準說明。

- **阻斷性（同交易，audit 失敗即回滾）**：`delete`、`permission_change`、`reveal_pii`、`conflict_resolve`、`manual_correction`。
- **非阻斷性（mutation 成功後寫入，失敗只記結構化 log）**：一般主檔的 `create`／`update`／`status_change`、匯入匯出的狀態留痕、已完成外部 side effect 的事後觀測。
- 依此判準，**據點／車輛／司機的 create 與 update 屬非阻斷性**，現行 `masterdata/app/audit.go` 的 `writeAuditBestEffort` 符合規範；`delete` 屬阻斷性。

| 失敗點 | 要求 |
|---|---|
| domain 驗證、版本檢查、必要 audit 失敗 | 整筆本地交易回滾，不回傳成功 |
| 交易提交回應遺失／逾時 | 結果可能未知；依作業識別碼查核，不盲目重送 mutation |
| 匯入重複提交 | 使用穩定作業鍵＋payload hash＋DB 唯一約束；同鍵不同內容拒絕，結果與寫入同交易保存 |
| 預覽後主檔／mapping 改變 | commit 時重新驗證與比對版本，不能信任 preview 即代表可寫入 |
| 多檔批次部分失敗 | 明列每檔／每列成功、失敗、待維護與原因；保留現行原子單位，不私自改成全批或逐列交易 |
| 通知失敗 | 不把已提交的主資料假稱回滾；持久化可重試狀態，明確顯示寄送未完成 |

匯入只支援 `.xlsx`。未對應欄位保存為可維護資料，不生成錯誤搭乘紀錄；pending 隔離、覆蓋匯入範圍及全 pending 行為依既有業務決策。冪等作業鍵如尚未存在，屬後續契約／schema 變更，不是本次已具備能力。

Domain event 僅在真有後續業務消費者時新增；普通 CRUD 不必發事件。需要「提交後保證可重試通知」時，才以同交易 outbox 保存 delivery intent，再由 worker 重試、去重與記錄失敗；只在記憶體發事件無法提供此保證。外部身分服務不與 PostgreSQL 共用交易，必須採可查核的操作狀態與補償／對帳流程；禁止把 SQL rollback 宣稱為撤銷外部操作。

## 7. 安全、查詢與契約

HTTP middleware 驗證身分並保留現有 `RequirePermission(module, action)`；application 對敏感 command 及資源範圍再次以 actor／policy port 驗證，讓 job 或其他入口不能繞過。domain 不讀 JWT。前端隱藏按鈕只改善體驗，不能取代後端授權。

營運查詢、歷史報表、待維護工作台各自使用明確 query intent，依系統規範處理 pending、inactive 與 soft delete；不能在共用 repository 加上一個「全站只查 active」造成歷史報表遺失。reporting 的跨 context SQL 投影可作為明確例外：只能讀取經擁有者確認的欄位／view，不寫別人的表、不匯入別人的 domain。每次提供端 schema 變更必須檢查投影依賴及隔離規則。

路由、envelope、錯誤碼、permission key、檔案格式與日期語意在搬移時維持不變。新契約才同步更新 API 規格、DTO、前端型別與範例；OpenAPI 須先確認對應實際端點，再產生型別，不能把過時規格當成自動正確。內部 error 保留 `errors.Is/As` 身分，transport 集中映射，SQL／stack／個資不進使用者訊息或稽核 snapshot。

## 8. 前端目標目錄與責任

```text
apps/web/src/
  app/                         main、router、layouts、providers、全域 styles
  pages/                       路由入口；組裝功能，處理跨功能畫面
    rides/RideCalendarPage.vue
  features/
    rides/
      api/                     DTO 與 endpoint functions
      queries/                 query keys、useQuery、useMutation
      model/                   表單／畫面型別、轉換、純驗證
      composables/             操作流程與 Vue lifecycle
      components/              功能元件，例如 RideCorrectionDrawer
      index.ts                 明確公開入口
    cases/
    driver-reports/
    sites/                     主檔可獨立成小功能，不共用萬用 CRUD store
    identity/
  shared/
    api/                       HTTP client、envelope、ApiError、generated types
    ui/                        DataTablePage、通用 dialog 等純 UI
    lib/                       formatters、純工具
```

同理新增 drivers、vehicles、caregivers、reports、exports、attendance、maintenance、holidays、notifications、audit 等功能，實際碰到才建目錄。前端 feature 依使用者工作劃分，與後端 context 不必一對一；一個匯出畫面可使用多個後端能力。

現行 `src/components/` 下有五個共用元件直接觸及 API 或 router，依本節定義不得留在 `shared/ui`，須隨對應 feature 遷移：`CaseSelectDialog.vue`（呼叫 `listAllCases`）、`cases/CaseCreateDialog.vue`（`createCase`＋兩個 `listAll`）、`masters/DriverCreateDialog.vue`（`createDriver`＋`listAllVehicles`）、`ImportPreviewDialog.vue`（自行 `import axios`）、`PrecheckResult.vue`（內用 `useRouter`）。

依賴只允許 `app -> pages -> features -> shared`，上層也可直接使用 shared。shared 不 import feature、page、router 或 auth store；feature 不 import page/app，不讀別的 feature 內部檔案。跨功能組裝由 page／app 完成，透過 props、emit、注入的 callback 或公開 API 交換資料；不以全域 event bus 隱藏呼叫鏈。身份提供者由 app 注入，其他 feature 不硬連 identity store。

page 只處理路由參數、版面與跨功能協調；功能 composable 處理操作流程；query 處理遠端資料生命週期；API function 處理 HTTP 與序列化。元件依責任拆分，不以固定行數或「每個按鈕一個檔案」機械切割。

## 9. 前端狀態與資料流

| 資料 | 擁有者 | 禁止做法 |
|---|---|---|
| API 列表、明細、loading/error、快取 | feature queries / TanStack Vue Query | 同一份列表再複製進 Pinia 當第二份真相 |
| 登入 session、使用者偏好、跨頁用戶端流程 | feature Pinia store，由 app 組裝 | shared HTTP client 直接 import store |
| 表單草稿、drawer 開關、當頁選取 | 元件或 composable 的 ref/reactive | 直接編輯 query cache 物件 |
| 可分享的分頁、篩選、月份 | route query，page 解析驗證 | URL 與 store 各維護不一致值 |
| 衍生欄位、統計顯示 | computed／純 mapper | 用 watch 再複製一份衍生 state |

現況：全 repo `defineStore` 只有 `stores/auth.ts` 一支，沒有把 API 列表複製進 Pinia 的情形，第一列的禁止事項目前沒有對應現況。真正的問題是重複全量抓取：`api/masters.ts` 的 `collectAllPages()`（`MAX_PAGE_SIZE = 100` 迴圈抓完所有頁）對外提供的 `listAllSites`／`listAllVehicles`／`listAllDrivers`，在 24 個 view／component 位置各自呼叫且完全無快取（`listAllVehicles` 單獨就有 14 個呼叫端）。這才是導入 TanStack Vue Query 的直接收益。

標準資料流：`page -> feature composable/query -> feature API -> shared HTTP -> Go API`；回應經 DTO mapper 進 query cache，表單以 copy 建立草稿，提交後才更新畫面。前端驗證提供即時提示，後端仍為業務合法性權威。

Query 規則：key 包含會影響結果的參數、月份、分頁及使用者／資料範圍，不放 token 或明文個資；登出、使用者切換與權限範圍變更清除相關快取。列表使用短期 staleTime，表單編輯期間背景重抓不能覆寫草稿。取消／忽略已過期請求，避免舊篩選回應覆蓋新畫面。查詢端點若有通知副作用，先拆出明確 mutation，才可接入自動重抓。

Mutation 預設不自動重試，重要寫入先等後端成功再呈現。成功後由 feature 失效自身列表／明細；跨 feature 的失效由 page/app 明確協調。例如更正搭乘後失效目前月曆、異常列表與相關月份報表。不要無差別清空所有資料，也不要遺漏統計快取。

409 保留草稿並引導重新載入／比對；不得自動改拿最新 version 重送以繞過衝突。逾時顯示結果待確認，先查核；匯入顯示檔案／列／欄位原因。所有流程都有 loading、empty、error 與可用的下一步。

共用 HTTP client 只負責 request、token provider、envelope／Blob error 解析及標準化 ApiError。401 透過 app 注入的 callback 去重處理登出與導頁；錯誤顯示由 feature 決定欄位錯誤或單次通知，app 提供未處理錯誤 fallback，不與 query retry 重複彈訊息。

Element Plus 與既有 Design Tokens 保留；不在這次架構遷移建立另一套 UI library。時間顯示沿用秒級 formatter；搬到 shared/lib 後保留舊 `@/utils/formatters` 單向 re-export 過渡。業務日期字串與 timestamp 分開處理，不能由瀏覽器時區自行改變服務日期。

## 10. 開發時如何遵循

1. 先判定所屬 context／feature、讀取相關業務規格，寫下使用者動作與成功／失敗條件。
2. 新業務規則先決定是 domain 不變條件或 application 協調；列出聚合、資料所有權、交易、版本及必要稽核。
3. 後端定義 command/query 與消費端 port，再實作 adapter 與 transport；bootstrap 注入。簡單查詢不強制穿過聚合。
4. 前端先定義 feature API／DTO、query key 與失效範圍，再實作表單 model、composable、component，最後由 page 組裝。
5. 檢查契約、permission key、日期、pending 隔離、錯誤碼及匯入／匯出語意；變更時同步對應規格。
6. 執行與實際 source logic 變更相符的驗證並記錄證據；未驗證的 DB、外部服務或 UI 行為分別列出。

Review 必答：規則是否能在不啟動 HTTP／DB 下測試？application 是否只看 port？是否有跨 context repository 直連？交易失敗是否真能撤回所有本地寫入？前端是否存在雙重狀態、shared 反向依賴或重複錯誤提示？

## 11. 從目前實作遷移

### 階段零：契約前置

expected version／409 的全站契約工程：SQL 更新條件、DTO 攜帶 expected version、`api.d.ts` 重新產生、前端所有編輯表單的 payload、409 的前端重新載入流程。全系統目前沒有任何一處實作（`grep "AND updated_at" apps/api/internal/modules/` 無命中），但階段二、三的完成條件都要求「版本衝突可驗證」。這是獨立於目錄重整的契約工程，與階段一並列為前置工作，不是階段二的附帶條件。

完成條件：至少一個端點（建議與階段二共用據點 CRUD）具備 expected version 檢查與 409 回應，前端有對應的重新載入流程可驗證。

### 階段一：建立可共存的規範與防護

先在實作任務中同步 AGENTS.md、`.agents/rules/development-rules.md`、backend skill 的 layering-rules、frontend skill 與 `internal/arch`。它們目前仍描述舊分層；本次僅新增目標文件，不表示自動防護已切換。

本階段是三項成本不對等的工作，分別驗收：

1. **後端 `arch_test.go` zone 模型改寫**：修補五個 fail-open 破口——`allowedInternal` 對 `to.kind()=="domain"` 無條件放行、`zoneOf` 對未知第一段回 `""` 導致 `internal/sharedkernel`／`internal/bootstrap` 不受掃描、`internalRoot=".."` 不掃描 `cmd/server`、`externalConfinement` 綁死 transport/infra 字面 kind、`isPersistencePackage` 以套件名 suffix 判斷。屬擴充既有測試，成本中等。
2. **四份規範文件命名同步**：`layering-rules.md`、`backend-architecture/SKILL.md`、`AGENTS.md`、`backend-framework.md` 改為新舊命名並列＋切換時點。
3. **前端 import 規則從零建立**：`apps/web` 目前沒有任何 eslint／dependency-cruiser 設定、沒有 lint script、非 npm workspace，這是新增一整套工具鏈（devDependency＋script＋CI），與第 1 項擴充既有測試不是同一量級。

架構檢查需同時識別舊與新 layout，拒絕未知目錄漏檢、module domain 反向依賴與跨 context import；不得增加 blanket allowlist 或放寬空 baseline。前端建立可執行 import 規則，驗證 shared 不依賴上層、features 不互讀內部檔案。這些防護是遷移工作的一部分，尚未存在。

完成條件：新舊目錄允許矩陣清楚、故意違規可被抓到、現有防護未失效；同步把每個 context 標記 legacy／migrating／target。

### 階段二：以據點 CRUD 建立完整切片

`masterdata/app/site_service.go` 的規則移至 domain 與 application；保留現有路由及 schema。將 SiteListView 與 masters API 中據點部分移至 sites feature，導入 Query 與 client injection。以此驗證 CRUD、查詢、授權、稽核及前端狀態分工，避免一開始便搬整個 ride。

完成條件：據點新增、編輯、狀態與刪除語意符合規格；依 §6 稽核等級判準，create／update 為非阻斷、delete 為阻斷，各自可驗證；版本衝突可驗證（依階段零的 expected version 實作）。切片可獨立回退，不影響其他主檔。本切片只驗證單一小型畫面（`SiteListView.vue` 384 行）的分工，不驗證大型畫面（如 1799 行的 `DriverReportImportView.vue`）的拆分方式。

### 階段三：以搭乘更正驗證 DDD

把 merge、人工更正與來源 fingerprint 相關不變條件歸入 ride domain；application 保留授權、鎖定、儲存及 audit。前端搬 RideCorrectionDrawer 與月曆資料協調，驗證更正後相關 query 失效。

完成條件：同車最新／跨車 OR、人工裁決保護、來源變動衝突、同交易回滾皆有相符證據；沒有直接繞過提供端 application 的新依賴。

### 階段四：匯入、個案與報表，再處理其餘功能

依賴順序為個案／排班公開 query → driverreport 匯入與 mapping → ride ingestion → reporting 投影與申報。caseimport 併回 casemgmt。最後整理 ops、identity、notification、audit 與 jobs；身份和外部副作用需各自規劃失敗狀態，不能用一般 CRUD 模板套過去。

完成條件：每個搬移功能保持對外契約與既有業務語意；匯入原子單位、待維護隔離、歷史報表可見度與非阻擋匯出均明確驗收。不是以「目錄搬完」判定完成。

### 階段五：移除過渡入口

所有呼叫端改用新入口後，刪除舊 app/infra/transport 與前端 re-export。更新現況 framework 文件及索引，將架構測試縮回唯一目標 layout；移除 legacy 路徑支援，baseline 持續空白。

完成條件：沒有雙寫、兩份業務演算法、循環 import 或過期文件指令。新開發者依本文即可找到正確責任位置。

### 回退界線

單一功能一次切換唯一寫入路徑，不雙寫新舊 repository。純程式碼搬移可回退切片；有 schema 變更時採 expand/contract，舊讀寫相容期結束前不得移除欄位。新增 version、唯一約束或新狀態後，回退必須檢查舊版相容性，不保證只 revert code 就安全。避免將演算法修改、資料回填與搬檔案混在同一批交付。

## 12. 驗證與完成標準

| 層級 | 應驗證內容 |
|---|---|
| Domain unit | 日期、合併、合法狀態、人工覆寫保護、不變條件 |
| Application unit | 使用案例分支、port 呼叫、授權、稽核失敗與錯誤傳遞；fake 不能證明 DB rollback |
| PostgreSQL integration | 同交易 rollback、unique／version 併發、鎖定、pending／刪除篩選、重送去重 |
| HTTP contract | 路由、DTO／envelope、status/error code、permission key、檔案格式 |
| Frontend unit／component | 草稿隔離、狀態轉換、query key／失效、401、409、匯入列錯誤 |
| Architecture | 新舊 layout、禁止依賴與未知區域辨識 |
| E2E | 使用者明確要求時，以真實本機 API／DB 驗證受影響流程；不以 mock 當整合證據 |

實作後依變更執行受影響 Go package 測試；前端在 `apps/web` 執行 type-check／build 與相關測試。需要元件測試時可導入 Vitest＋Vue Test Utils，但現有 package scripts 尚不是此組合，須另做相容性與設定工作。不要直接假設工具已安裝。

現行 `test` script 是 Node 內建 test runner（`node --experimental-strip-types --test`，9 支 unit test），導入 Vitest 會與它形成兩套 runner，需先決定併軌或取代，不要兩套並存。`gen:types` 目前把 openapi 型別輸出到 `src/types/api.d.ts`；generated types 搬到 `shared/api/` 時，要同步改這支 script 的輸出路徑，否則型別產生會回寫到已廢棄的位置。

本次為文件規劃，沒有修改 application source、安裝套件、執行 migration、測試、build、E2E 或部署。後續交付需分別記錄靜態、單元、整合與 runtime 證據。

## 13. 設計參考

以下為概念與工具依據；本文件的目錄、context 切分與遷移順序是依本專案需求做出的設計，並非官方規定。

- [Robert C. Martin：The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)：依賴朝內、外部資料格式不進入內層。
- [Martin Fowler：Bounded Context](https://martinfowler.com/bliki/BoundedContext.html)：用清楚邊界維持模型語意，並描述 context 間關係。
- [Vue 官方：State Management](https://vuejs.org/guide/scaling-up/state-management)：Vue 的狀態管理與 Pinia 建議。
- [Pinia 官方介紹](https://pinia.vuejs.org/introduction.html)：跨元件／頁面的共享狀態。
- [TanStack Vue Libraries](https://tanstack.com/libraries/vue)：Vue Query 提供 server-state 與資料取得能力。
