---
name: development-guidelines
description: Read before every development, fix, refactor, or test change in this repository; establishes the shared project workflow and routes only to applicable domain guidelines.
---

# Project development guidelines

這是每次開發工作的第一個 skill。它定義跨領域共通規範；API、資料寫入、權限、migration、部署與業務資料語意由專項 skill 定義，依實際變更範圍讀取。

## 開始工作前

1. 記錄 `git status --short --branch` 與 `git rev-parse HEAD`，保留既有未提交修改，確認目前分支與實際工作樹是本次工作的現況。
2. 寫下本次目標、非目標、受影響的入口與檔案範圍；涉及跨層功能時追蹤：

   ```text
   UI / client / mock -> HTTP route + DTO -> application / domain
                         -> repository / adapter -> database / provider
   ```

3. 依入口檔案與變更效果讀取專項 skill；不要因為同一功能同時存在前後端，就載入與本次範圍無關的規範。
4. 把 `static inspection`、`hypothesis`、`automated test` 與 `runtime proof` 分開記錄；規劃或設定檔不等同於實際執行證據。

## 變更規範

- 每項契約與規則只保留一個 canonical source；修改後同步檢查直接消費者、型別、測試、mock 與必要文件。
- 保留空值、零值、未知值與錯誤的業務語意；未提供與非法輸入分別處理，明確錯誤優先於靜默替代結果。
- 行為變更與機械格式化分開；若同時存在，先檢查 diff 範圍與可審查性。
- 採用與變更範圍相稱的 focused validation；回報每項檢查的實際結果，未執行的 runtime 行為標示為未驗證。

## 完成條件

- 已列出受影響的入口、上下游消費者與適用專項 skill。
- 已依專項 skill 完成契約、資料、權限、部署或介面檢查。
- 驗證結果可對應到實際指令、測試或靜態證據。

## 歷史依據

只有在檢視 commit 歷史、維護本組規範或需要追查規則來源時，才讀 [references/history-evidence.md](references/history-evidence.md)。
