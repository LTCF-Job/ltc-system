import { type Page, expect } from '@playwright/test'

export type TestRole = 'admin' | 'viewer'

export const DEMO_ROLE_LABELS: Record<TestRole, string> = {
  admin: '系統管理員',
  viewer: '檢視者'
}

export async function loginAs(page: Page, role: TestRole = 'admin', targetPath?: string) {
  const loginUrl = targetPath && targetPath !== '/' ? `/login?redirect=${encodeURIComponent(targetPath)}` : '/login'
  await page.goto(loginUrl)
  await page.waitForLoadState('networkidle')

  // 使用展示模式快速按鈕切換身分
  const roleBtn = page.getByRole('button', { name: new RegExp(DEMO_ROLE_LABELS[role]) })
  if (await roleBtn.isVisible()) {
    await roleBtn.click()
    await page.waitForLoadState('networkidle')
  } else {
    // 降級由 localStorage 寫入 session
    await page.evaluate((r) => {
      localStorage.setItem('ltc_auth_token', `demo_token_${r}`)
      localStorage.setItem(
        'ltc_auth_user',
        JSON.stringify({
          id: `demo_${r}`,
          name: `測試${r}`,
          role: r,
          email: `${r}@example.com`
        })
      )
    }, role)
    if (targetPath) {
      await page.goto(targetPath)
    } else {
      await page.goto('/')
    }
    await page.waitForLoadState('networkidle')
  }

  // 確保在目標頁面
  if (targetPath && targetPath !== '/') {
    if (!page.url().includes(targetPath)) {
      await page.goto(targetPath)
      await page.waitForLoadState('networkidle')
    }
  }
}

export async function logout(page: Page) {
  // 開啟使用者下拉選單並點選登出
  const userDropdown = page.locator('.user-dropdown-link').first()
  if (await userDropdown.isVisible()) {
    await userDropdown.click()
    const logoutItem = page.getByText('登出系統').last()
    await logoutItem.click()
  } else {
    await page.evaluate(() => {
      localStorage.removeItem('ltc_auth_token')
      localStorage.removeItem('ltc_auth_user')
    })
    await page.goto('/login')
  }
  await page.waitForURL('**/login', { timeout: 10000 })
}
