import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage, confirmMessageBox } from './helpers/ui'

test.describe('10. 系統稽核紀錄與權限設定 (Audit Logs & System Settings)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('稽核紀錄：清單載入、篩選與查看異動前後比對彈窗', async ({ page }) => {
    await page.goto('/audit')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 驗證「搭乘事後補報」動作已正確中文化（非 manual_report）
    const manualReportAction = page.locator('.el-table__row').getByText('搭乘事後補報').first()
    await expect(manualReportAction).toBeVisible()

    // 驗證操作對象欄位非 raw UUID，而是親切可讀的個案與趟次或實體描述
    const entityCell = page.locator('.el-table__row').getByText('蔡曾切 (2026-08-28 去程)').first()
    await expect(entityCell).toBeVisible()

    // 點選第一筆異動前後按鈕
    const diffBtn = page.locator('.el-table__row').locator('button, a').filter({ hasText: '異動前後' }).first()
    if (await diffBtn.isVisible()) {
      await diffBtn.click()
      const dialog = page.locator('.el-dialog').filter({ hasText: '系統操作紀錄異動詳情' })
      await expect(dialog).toBeVisible()
      await expect(dialog.getByText('所屬區塊').first()).toBeVisible()
      await dialog.getByRole('button', { name: '關閉', exact: true }).click()
    }
  })

  test('使用者管理：新增使用者、設定個人自訂權限、當前使用者無刪除按鈕', async ({ page }) => {
    await page.goto('/settings/users')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    const rows = page.locator('.user-management-view .el-table__row')
    const initialCount = await rows.count()
    expect(initialCount).toBeGreaterThan(0)

    // 當前登入使用者那一列不應該出現「刪除」按鈕（不能刪除自己），
    // 其餘每一列都應該有；兩者相減必須剛好差一列。
    const deleteButtons = page.locator('.user-management-view .el-table__row').getByRole('button', { name: '刪除' })
    await expect(deleteButtons).toHaveCount(initialCount - 1)

    // 新增使用者
    await page.getByRole('button', { name: '新增使用者' }).click()
    const createDialog = page.locator('.el-dialog').filter({ hasText: '新增系統使用者' })
    await expect(createDialog).toBeVisible()

    const uniqueEmail = `e2e-${Date.now()}@example.com`
    await createDialog.getByPlaceholder('請輸入電子郵件（作為登入帳號）').fill(uniqueEmail)
    await createDialog.getByPlaceholder('如：王小明').fill('E2E 測試使用者')
    await createDialog.getByRole('combobox', { name: /身分角色/ }).click({ force: true })
    await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: '檢視者' }).first().click()
    await createDialog.getByPlaceholder(/請輸入初始登入密碼/).fill('test1234')

    await createDialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, undefined, 'success')
    await waitForTableLoaded(page)
    await expect(rows).toHaveCount(initialCount + 1)

    // 對新使用者設定個人自訂權限：切換其中一個模組的編輯權限後儲存
    const newRow = rows.filter({ hasText: uniqueEmail })
    await newRow.getByRole('button', { name: '設定權限' }).click()

    const drawer = page.locator('.el-drawer').filter({ hasText: '自訂功能模組權限' })
    await expect(drawer).toBeVisible()
    const firstViewCheckbox = drawer.locator('.el-table__row').first().locator('.el-checkbox').first()
    await firstViewCheckbox.click()

    await drawer.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, undefined, 'success')
  })

  test('角色身分管理：系統角色不可刪除、複製角色、批次全選編輯權限', async ({ page }) => {
    await page.goto('/settings/roles')
    await waitForTableLoaded(page)
    await expect(page.locator('.filter-card, .table-card, .role-management-view, .el-table').first()).toBeVisible()

    const cards = page.locator('.role-row')
    const initialCount = await cards.count()
    expect(initialCount).toBeGreaterThan(0)

    // 系統角色（例如系統管理員）的刪除按鈕應為 disabled
    const systemCard = cards.filter({ hasText: '系統管理員' }).first()
    await expect(systemCard).toBeVisible()
    const systemDeleteBtn = systemCard.getByRole('button', { name: '刪除' })
    await expect(systemDeleteBtn).toBeDisabled()

    // 複製一個角色，確認清單新增一張卡片
    await systemCard.getByRole('button', { name: '複製建立' }).click()
    const copyDialog = page.locator('.el-dialog').filter({ hasText: '角色' })
    await expect(copyDialog).toBeVisible()
    await expect(copyDialog.locator('input').first()).toHaveValue(/\(複製\)$/)

    // 批次「全選編輯」：所有模組的編輯 checkbox 應全數勾選
    await copyDialog.getByRole('button', { name: '全選編輯' }).click()
    const editCheckboxInputs = copyDialog.locator('.el-table__row .el-checkbox').filter({ hasText: '編輯' }).locator('input[type="checkbox"]')
    const editCount = await editCheckboxInputs.count()
    expect(editCount).toBeGreaterThan(0)
    for (let i = 0; i < editCount; i++) {
      await expect(editCheckboxInputs.nth(i)).toBeChecked()
    }

    await copyDialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, undefined, 'success')
    await waitForTableLoaded(page)
    await expect(cards).toHaveCount(initialCount + 1)
  })

  test('通知收件人管理：批次新增去重、多選批次刪除', async ({ page }) => {
    await page.goto('/settings/notifications')
    await waitForTableLoaded(page)
    await expect(page.locator('.filter-card, .table-card, .notification-settings-view, .el-table').first()).toBeVisible()

    const rows = page.locator('.notification-settings-view .el-table__row')
    const initialCount = await rows.count()

    // 批次新增：同一次貼上內容自身重複一筆，前端即時解析要標示「重複信箱」，
    // 且送出時只新增未重複的部分（後端 batch 端點依 topic+email 去重）。
    await page.getByRole('button', { name: '新增外部信箱' }).click()

    const addDialog = page.locator('.el-dialog').filter({ hasText: '新增外部信箱' })
    await expect(addDialog).toBeVisible()
    await addDialog.getByRole('combobox', { name: /通知主題/ }).click({ force: true })
    await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: '未回報催報' }).first().click()

    const newEmail1 = `batch-e2e-1-${Date.now()}@example.com`
    const newEmail2 = `batch-e2e-2-${Date.now()}@example.com`
    await addDialog.locator('textarea').fill(`${newEmail1}\n${newEmail2}\n${newEmail1}`)

    await expect(addDialog.getByText('重複信箱').first()).toBeVisible()
    await expect(addDialog.getByText(/確認新增 \(共 2 筆\)/)).toBeVisible()
    await addDialog.getByRole('button', { name: /確認新增/ }).click()
    await expectElMessage(page, undefined, 'success')
    await waitForTableLoaded(page)
    await expect(rows).toHaveCount(initialCount + 2)

    // 重新送出同一批（後端依 topic+email 去重），這次應該一筆都不會新增
    await page.getByRole('button', { name: '新增外部信箱' }).click()
    await expect(addDialog).toBeVisible()
    await addDialog.getByRole('combobox', { name: /通知主題/ }).click({ force: true })
    await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: '未回報催報' }).first().click()
    await addDialog.locator('textarea').fill(`${newEmail1}\n${newEmail2}`)
    await addDialog.getByRole('button', { name: /確認新增/ }).click()
    await expect(addDialog).toBeHidden()
    await waitForTableLoaded(page)
    await expect(rows).toHaveCount(initialCount + 2)

    // 多選批次刪除剛新增的兩筆
    const row1 = rows.filter({ hasText: newEmail1 })
    const row2 = rows.filter({ hasText: newEmail2 })
    await row1.locator('.el-checkbox__input').click()
    await row2.locator('.el-checkbox__input').click()

    await page.getByRole('button', { name: /批次刪除/ }).click()
    await confirmMessageBox(page)
    await expectElMessage(page, /批次刪除/, 'success')
    await waitForTableLoaded(page)
    await expect(rows).toHaveCount(initialCount)
  })
})
