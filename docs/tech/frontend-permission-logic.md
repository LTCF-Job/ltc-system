# 前端權限判斷邏輯

`frontend-pages.md` 列的「允許角色」只是路由 `meta.roles` 陣列，方便快速掃過各頁大致給哪些角色用；但**實際的權限檢查幾乎都不是走這個陣列**。這份文件寫的是路由守衛實際跑的判斷順序，改權限規則前一定要看這份，不要只改 `meta.roles` 以為就生效了。

## 路由守衛的判斷順序（`router/guards.ts`）

`beforeEach` 依序檢查，任一步驟決定放行／攔截後就不看後面的步驟：

1. 這個路由不是 `meta.public` 且使用者未登入 → 導去 `/login`。
2. 已登入卻要進 `/login` → 導去首頁。
3. **`meta.module` 有設定值時**（目前所有非公開路由都有設）→ 呼叫 `authStore.hasPermission(module, 'view')`，**不看 `meta.roles`**，沒權限就導回首頁並跳警告訊息。
4. 只有 `meta.module` 沒設定時，才會退回看 `meta.roles`，呼叫 `authStore.can(roles)`。

也就是說：目前 `router/index.ts` 每個路由的 `meta.roles` 陣列，實務上只是文件性質的標註（給人看大概哪些角色會用到），真正決定「這個角色進不進得去這頁」的是 `authStore.hasPermission()` 背後那一整套模組權限表。改一個頁面能不能被哪個角色看到，要改的是下面這套邏輯，不是路由表的 `roles`。

## 模組權限表（`src/types/domain.ts`）

系統把每個功能頁面定義成一個「模組」（`SYSTEM_MODULES`，例如 `masters_cases`、`rides_calendar`、`settings_users`），每個模組對每個角色有一組 `{ view: boolean, edit: boolean }`：

- `DEFAULT_ROLE_PERMISSIONS.admin`：所有模組 `view/edit` 全部 `true`（用程式產生，不用一個個列）。
- `DEFAULT_ROLE_PERMISSIONS.dispatcher` / `.staff`：手動列出每個模組的權限，兩者目前設定幾乎一樣（都能看能編大部分業務模組），差異在 `dispatcher`／`staff` 這兩個角色本身在後端 `RequireRoles` 裡是分開判斷的，但前端權限表給的預設值目前相同。
- `DEFAULT_ROLE_PERMISSIONS.driver`：只能看／編「車輛維修保養」「出勤與油資」跟看「搭乘月曆」「車輛主檔」「司機主檔」，個案、單位、表單、報表、匯出、系統設定全部沒有權限。
- `DEFAULT_ROLE_PERMISSIONS.viewer`：除了 `audit_logs`、`settings_users`、`settings_roles` 這三個系統管理類模組看不到，其他模組都只有 `view`，沒有 `edit`。

加一個新頁面（新模組）時，記得同時做三件事：`SYSTEM_MODULES` 加一筆、每個角色的 `DEFAULT_ROLE_PERMISSIONS` 補上這個模組的 `view/edit`、路由 `meta.module` 指到這個新模組 id——三個地方漏一個都會導致權限判斷不如預期（漏了模組定義會被 `hasPermission` 當成沒權限，因為 `modPerm` 找不到直接回傳 `false`）。

## 個人自訂權限覆蓋（`effectivePermissions`）

「角色身分管理」頁允許在角色預設值之外，對單一使用者再設定 `customPermissions`（存在 `UserDTO.customPermissions`）。實際生效的權限是 `stores/auth.ts` 的 `effectivePermissions` computed：

```
effectivePermissions = roleDefault 複製一份
                        再用 user.customPermissions 逐模組覆蓋（有設定該模組才覆蓋，沒設定的模組維持角色預設）
```

覆蓋是**整個模組物件替換**（`{ view, edit, delete }` 一起換掉），不是欄位層級合併——自訂權限只設了 `edit: true` 也會連 `view`／`delete` 一起被覆蓋成該筆自訂資料裡的值，寫自訂權限資料時三個欄位都要給值，不能只給其中一個期待另一個沿用角色預設。

`hasPermission(module, action)` 最終看的就是這個合併後的結果，admin 角色永遠 bypass（不查表，直接 `true`）。

## 角色階層（`can()`，用在沒有 `module` 的少數判斷）

```
ROLE_HIERARCHY = { admin: 4, dispatcher: 3, staff: 3, driver: 2, viewer: 1 }
```

`can(requiredRoles)`：admin 永遠 `true`；否則比較「自己角色的等級」是否 ≥ 陣列中任一個要求角色的等級（不是要求完全命中角色名稱，是等級比較，`dispatcher` 跟 `staff` 同等級 3，兩者互相都能通過對方的門檻）。這套階層邏輯目前只在 `meta.roles`（沒設 `module` 時）跟少數元件內部判斷使用，跟上面的模組權限表是兩套獨立機制，不要混著改。

## 跟後端授權的對應關係

前端這套「模組 view/edit/delete」矩陣，**角色層級**已經跟後端對齊：後端 `auth.RequirePermission(module, action)` 直接查角色目前的 `roles.permissions`（同一份資料，「角色身分管理」頁存的就是它），不再是路由層級的粗粒度角色字串白名單，詳見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md)。`/users`、`/roles`、`/auth/change-password`、`/demo/reset`、`/tasks/*`、`/holidays*` 仍是 `RequireRoles` 白名單，不受角色矩陣控制（理由見該決策文件）。

## ⚠️ 個人自訂覆蓋（`customPermissions`）仍是已知落差

上面「角色層級」已對齊，但「使用者管理」頁對**單一使用者**再疊加的 `customPermissions` 覆蓋，後端目前**沒有**讀取——JWT 的 `setActorFromClaims` 沒有解析 `app_metadata.custom_permissions`，`auth.RequirePermission` 查的是角色矩陣，不是這個使用者的個人覆蓋。

實際影響：如果透過「使用者管理」頁把某個 `staff` 使用者的 `masters_cases.edit` 自訂關掉，前端會隱藏／擋掉編輯按鈕，**但後端仍照這個使用者的角色（`staff`）矩陣放行**，只要繞過前端直接打 `PATCH /cases/:id` 一樣會成功——後端目前沒有能力執行個人層級的細粒度限制，只有角色層級。這是設計上先天的落差，不是 bug，但接手的人要知道個人自訂權限只是 UX 層面的引導，不是安全邊界；真正的安全邊界是使用者所屬的角色矩陣。
