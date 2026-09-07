# Commit history evidence

這份檔案是規範的來源索引，不是每次開發都要讀取的操作清單。

截至 2026-09-06 盤點快照：`HEAD=b8e0b2b`、`HEAD` 可達 238 個 commit、所有 refs 共 256 個 commit。完整分析以 `git log --all`、各 commit diff 與目前程式碼交叉比對；下表只保留能形成專案規範的重複模式。

| 規範領域 | 代表 commit | 歷史觀察 | 形成的專案規範 |
|---|---|---|---|
| 跨層交付 | `6660c4e`、`0c71760`、`398c82d`、`811a3b8`、`99a8c50`、`62e17ab`、`3b8b136` | 大型前後端與 Mock 批次交付後才補串接、envelope 與錯誤處理 | 先固定 execution path 與契約，再逐入口驗證 |
| API 契約 | `c35e71e`、`24c5620`、`61dec37`、`fb59617`、`46f4401`、`3070531`、`a8089e2`、`f0f10fa`、`d29b900`、`b16eaeb`、`4bde7e2`、`c81878b` | response envelope、URL ID、欄位名稱、錯誤與原始 payload 曾在上下游漂移 | path ID canonical、單一 envelope 邊界、同步更新所有 consumer |
| Mock／Demo 邊界 | `1694da6`、`0fdc8a9`、`f6cc053`、`641b5f9`、`514b741`、`5ab88dc` | 啟用條件分散，正式錯誤曾回填假資料，未命中 API 未暴露 | 明確 data plane、生命週期與 fail-closed 行為 |
| Excel／XLSX | `aad94f1`、`fd9efee`、`c90fa2a`、`7c57184`、`858043d`、`79243fd`、`ebe06b3` | 範本、欄位位置、Mock 與 parser 分開演進 | golden fixture、parser round-trip、HTTP body 與 mock parity |
| 寫入與稽核 | `3bdbcf2`、`dce6108`、`c136dc3`、`c0f9192`、`412b5f3` | 多重寫入未共用 transaction、audit error 被忽略、查詢失敗被當成空資料 | 明確 mutation boundary、同一 tx context、audit policy |
| Auth／權限 | `0f02b78`、`6ceb030`、`843c85e`、`c0f9192` | JWT claim、route、權限矩陣、migration 與 cache 未同步 | identity boundary、route-module-action-migration 一致性 |
| Migration | `3bdbcf2`、`7edb80e`、`85ef428`、`843c85e` | 合併與重構後出現重複版本或漏套用權限 migration | 唯一版本、up/down、clean／existing DB replay |
| 部署 | `6519292`、`c655967`、`ae51d8a`、`8c7c192`、`f5e31aa`、`cc444ac`、`77bed1a` | trigger、checkout ref、root、env 與 secret scope 曾漂移 | exact commit、environment matrix、部署後 runtime proof |
| 業務資料語意 | `06493fc`、`e38a764`、`c8c309d`、`7273722`、`962a62c`、`026cb2b`、`bc4dca2`、`cb6eb5b`、`d79c4ac`、`15da3f1`、`b8e0b2b` | 空值預設、民國日期、排班、假日、清除、UUID array 與請求邊界語意曾分散 | domain parser／rule table 與邊界案例 |
| UI 契約 | `97cf3eb`、`8b41ecb`、`73b0057`、`eb835ac`、`12b467d`、`f1e4c7d` | 共用 table、token 與頁面重構未一次盤點 | 先列出受影響頁面，使用共用 component／token |
| Review 可讀性 | `3070531` | 行為修正混入大量 TypeScript formatter 變更 | 行為與機械格式化分開 |

## 不形成一般開發規範的個案

- Docker Hub TLS timeout 是環境／網路故障，依 deployment 的 operational diagnosis 處理。
- Git guard 阻擋 commit 是本機授權或工具狀態；回報 commit 未建立即可，不延伸成 application 規範。
- 單次文案或單一 CSS 數值只留在該次 review；只有跨頁重複模式才提升為共用 UI 規範。

## 文件缺口

入口與 workflow 曾指向 `docs/architecture.md`，目前架構資訊實際分散於 `docs/tech/backend-framework.md`、`docs/tech/frontend-framework.md` 與 `docs/tech/README.md`。這是待另案處理的文件 routing 缺口，本次不捏造缺失文件內容。
