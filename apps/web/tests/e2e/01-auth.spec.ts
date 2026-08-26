import { test, expect } from '@playwright/test'
import { loginAs, logout } from './helpers/auth'

test.describe('01. 認證與權限控制 (Authentication & Authorization)', () => {
  test('登入介面渲染完整且包含展示快速身分切換按鈕', async ({ page }) => {
    await page.goto('/login')
    await expect(page).toHaveTitle(/長照交通接送/)
    await expect(page.getByRole('heading', { name: '長照交通接送後台系統' })).toBeVisible()
    await expect(page.getByPlaceholder('請輸入電子郵件')).toBeVisible()
    await expect(page.getByPlaceholder('請輸入密碼')).toBeVisible()
    await expect(page.getByRole('button', { name: '登入系統' })).toBeVisible()

    // 檢查展示模式按鈕
    await expect(page.getByRole('button', { name: /系統管理員/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /調度員/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /司機/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /檢視者/ })).toBeVisible()
  })

  test('展示模式快速切換為 Admin 身分可成功進入儀表板', async ({ page }) => {
    await loginAs(page, 'admin')
    await expect(page.getByRole('menuitem', { name: '總覽儀表板' })).toBeVisible()
    await expect(page.getByText('在案個案總數')).toBeVisible()
  })

  test('Viewer 角色權限受限，無法存取管理員專屬的系統設定頁面', async ({ page }) => {
    await loginAs(page, 'viewer')
    // 嘗試直接導向使用者管理頁面
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')
    const currentUrl = page.url()
    expect(currentUrl).not.toContain('/settings/users')
  })

  test('登出系統後狀態被清除並安全導回登入頁面', async ({ page }) => {
    await loginAs(page, 'admin')
    await logout(page)
    await expect(page.getByRole('button', { name: '登入系統' })).toBeVisible()
  })
})

