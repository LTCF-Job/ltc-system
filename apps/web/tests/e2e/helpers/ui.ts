import { type Page, expect } from '@playwright/test'

/**
 * 等待 Element Plus 成功/警告/錯誤 Message 提示框出現
 */
export async function expectElMessage(
  page: Page,
  textOrRegex?: string | RegExp,
  type: 'success' | 'warning' | 'error' | 'info' = 'success'
) {
  // Element Plus 訊息淡出前仍留在 DOM，可能同時存在多則，一律只看最新一則。
  const messageLocator = (
    textOrRegex
      ? page.locator(`.el-message--${type}`).filter({ hasText: textOrRegex })
      : page.locator(`.el-message--${type}`)
  ).last()
  await expect(messageLocator).toBeVisible({ timeout: 10000 })
}

/**
 * 自動點選 Element Plus MessageBox 的確認按鈕
 */
export async function confirmMessageBox(page: Page) {
  const msgBox = page.locator('.el-message-box')
  await expect(msgBox).toBeVisible({ timeout: 5000 })
  const confirmBtn = msgBox.getByRole('button', { name: /確定|確認|是|刪除/ })
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

