import { type Page, expect } from '@playwright/test'

/**
 * 等待 Element Plus 成功/警告/錯誤 Message 提示框出現
 */
export async function expectElMessage(
  page: Page,
  textOrRegex?: string | RegExp,
  type: 'success' | 'warning' | 'error' | 'info' = 'success'
) {
  const messageLocator = page.locator(`.el-message--${type}`)
  await expect(messageLocator).toBeVisible({ timeout: 10000 })
  if (textOrRegex) {
    await expect(messageLocator).toContainText(textOrRegex)
  }
}

/**
 * 自動點選 Element Plus MessageBox 的確認按鈕
 */
export async function confirmMessageBox(page: Page) {
  const msgBox = page.locator('.el-message-box')
  await expect(msgBox).toBeVisible({ timeout: 5000 })
  const confirmBtn = msgBox.getByRole('button', { name: /確定|確認|是/ })
  await confirmBtn.click()
}

/**
 * 於 Element Plus 下拉選單中選取特定項目
 */
export async function selectElOption(
  page: Page,
  placeholderOrLabel: string,
  optionLabel: string
) {
  const select = page.getByPlaceholder(placeholderOrLabel).or(page.locator(`.el-select`).filter({ hasText: placeholderOrLabel }))
  await select.click()
  const dropdown = page.locator('.el-select-dropdown:visible')
  await dropdown.locator('.el-select-dropdown__item').filter({ hasText: optionLabel }).first().click()
}

/**
 * 等待頁面表格載入指示器結束
 */
export async function waitForTableLoaded(page: Page) {
  await page.waitForLoadState('networkidle')
  const loadingMask = page.locator('.el-loading-mask:visible')
  if (await loadingMask.count() > 0) {
    await loadingMask.first().waitFor({ state: 'hidden', timeout: 10000 })
  }
}

