---
name: migration-guidelines
description: Use when changing SQL migrations, database schema, indexes, constraints, seed data, role or permission data, migration ordering, or database versioning.
---

# Database migration guidelines

## 版本與內容

- 每個已發佈 migration 使用唯一、遞增且可追蹤的版本號；合併分支前檢查平行變更是否使用相同版本。
- migration 的 `up` 與 `down` 成對維護；已發佈版本的檔名與版本不以事後改名掩蓋歷史錯誤，修正另加新 migration。
- schema、index、constraint、seed、角色與 permission matrix 視為同一資料契約的一部分；修改其中一項時同步檢查應用程式與權限測試。

## 驗證與交付

- 在乾淨資料庫從頭 replay，也在代表性的既有資料庫上套用；檢查資料保留、rollback、constraint 與重複執行語意。
- 新增欄位、索引或權限資料時驗證舊資料、既有查詢、seed idempotency 與應用程式版本相容性。
- deployment job 使用與應用程式相同的 commit、environment、database URL 與 secret scope；workflow 設定正確不等於 migration 已在目標環境成功執行。
- 回報時區分 SQL 靜態檢查、local replay、CI 結果與目標環境 runtime proof。
