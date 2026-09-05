---
name: auth-permission-guidelines
description: Use when changing JWT validation, login, actor identity, roles, permissions, protected routes, permission cache, user administration, or authorization-related audit data.
---

# Authentication and permission guidelines

## Identity boundary

- 業務角色與 data-plane 權限以伺服器端驗證後的 JWT `app_metadata` 為準；token 內的可由使用者自行修改欄位不作為權威來源。
- 驗證 issuer、audience、expiration 與 subject；無法解析或缺少必要 actor identity 的請求進入明確的 authentication error。
- 正式 Supabase Auth、demo login、fixture 與 offline mode 分開定義資料平面、啟用條件與生命週期；正式流程不共享 demo 權限旁路。

## Permission contract

- 每條受保護 route 都對應 route、module、action 與角色矩陣；新增或修改任一項時，同步檢查 middleware、route test、角色 seed／migration、前端權限判斷與決策文件。
- 權限判斷集中使用同一 permission service／policy，不在頁面、handler 與 mock 各自複製規則。
- 權限或角色更新後立即使相關 cache invalidation；cache 命中不能讓已撤銷的權限持續有效。

## Audit and verification

- audit snapshot 採最小必要資料並遮罩 PII、token、secret 與不應外洩的 claim。
- 測試 valid／expired／wrong issuer／wrong audience／missing subject token、角色矩陣每個結果、拒絕未授權 route、權限撤銷後 cache 行為與 migration seed 完整性。
