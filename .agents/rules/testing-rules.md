# 測試執行規則 (Testing Rules)

## 核心原則
- **執行測試時機**：只有在實際變更「應用程式原始碼（application source code logic）」時，才需要執行對應的自動化測試（如前端 E2E 測試、後端單元測試）。
- **略過測試時機（Bypass）**：以下與程式碼業務邏輯無關的變更或任務，一律不跑測試：
  - 文件（Documentation，如 Markdown 文件、說明文件）
  - 指令與腳本（Scripts / Commands）
  - 測試檔案本身（Test files / Fixtures）
  - 註解（Comments）
  - 設定檔（Configurations）、環境變數範本
  - 除錯排查、規劃討論、Review、問答與翻譯等非原始碼邏輯任務
