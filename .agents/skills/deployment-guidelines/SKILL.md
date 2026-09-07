---
name: deployment-guidelines
description: Use when changing CI/CD workflows, Dockerfiles, deployment scripts, Vercel, Cloud Run, environment variables, secrets, build roots, or production release checks.
---

# Deployment guidelines

## 來源與環境

- test、build 與 deploy 對同一個明確 commit 執行；workflow 的 trigger、checkout ref、branch filter 與 deploy ref 必須一致。
- 明確固定 repository root、app root、build output、runtime command 與 environment；環境名稱大小寫與 secret／variable scope 按平台實際名稱核對。
- migration、seed、build 與 application release 的先後順序寫入 workflow 契約，並確認失敗會阻止不完整版本發布。
- Docker、Vercel、Cloud Run 與其他 provider 的設定以實際執行環境為準；YAML 可解析或本機 build 成功只屬於靜態／local 證據。

## 交付驗證

- 部署後檢查實際 revision、health endpoint、關鍵 API、資料庫版本與必要的登入／權限路徑。
- 將 provider log、部署輸出、目標環境 request 結果與本機測試分開記錄；未連到目標環境時標示 runtime proof 未完成。
- Docker Hub TLS、proxy、DNS 或 registry timeout 屬於 operational diagnosis；先驗證執行環境，再決定是否需要修改程式或 Dockerfile。
