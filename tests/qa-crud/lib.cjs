const path = require('node:path')
const { chromium } = require(path.join(__dirname, '../../apps/web/node_modules/playwright'))

const BASE = process.env.QA_BASE_URL || 'http://localhost:3000'
// 同一次執行共用的資料標記，讓測試資料可被辨識與清除，重跑也不會撞唯一鍵。
const TAG = process.env.QA_TAG || 'QA' + String(Date.now()).slice(-6)
const LOGIN_EMAIL = process.env.QA_LOGIN_EMAIL || 'admin@example.com'
const LOGIN_PASSWORD = process.env.QA_LOGIN_PASSWORD || 'pw123456'

// 回應 envelope 的 data 內若出現大寫開頭的 key，代表 transport 直接回傳了 domain model。
async function pascalCaseKeys(response) {
  if (!/\bapplication\/json\b/.test(response.headers()['content-type'] || '')) return []
  let payload
  try { payload = JSON.parse(await response.text()) } catch (_) { return [] }
  const found = new Set()
  const walk = (node, depth) => {
    if (depth > 4 || node === null || typeof node !== 'object') return
    if (Array.isArray(node)) { node.slice(0, 5).forEach(v => walk(v, depth + 1)); return }
    for (const [key, value] of Object.entries(node)) {
      if (/^[A-Z]/.test(key)) found.add(key)
      walk(value, depth + 1)
    }
  }
  walk(payload?.data, 0)
  return [...found]
}

async function launch({ headless = process.env.QA_HEADED !== '1' } = {}) {
  const browser = await chromium.launch({ headless })
  const ctx = await browser.newContext({ viewport: { width: 1680, height: 1050 }, acceptDownloads: true })
  const page = await ctx.newPage()
  const net = { failed: [], console: [], pageerror: [], calls: [] }

  page.on('console', m => { if (m.type() === 'error') net.console.push(m.text().slice(0, 300)) })
  page.on('pageerror', e => net.pageerror.push(String(e).slice(0, 300)))
  page.on('request', r => {
    if (!r.url().includes('/api/v1/')) return
    if (!['POST', 'PATCH', 'PUT', 'DELETE'].includes(r.method())) return
    let body = ''
    try { body = (r.postData() || '').slice(0, 900) } catch (_) { body = '(binary)' }
    net.calls.push({ phase: 'req', method: r.method(), url: r.url().replace(BASE, ''), body })
  })
  page.on('response', async r => {
    if (!r.url().includes('/api/v1/')) return
    const method = r.request().method()
    const url = r.url().replace(BASE, '')

    if (r.status() < 400) {
      // domain model 直接當回應回傳會輸出 PascalCase，前端讀不到卻不會報錯。
      // 這條掃過所有路由，新模組自動涵蓋，比每個模組各補一支測試划算。
      const offenders = await pascalCaseKeys(r)
      if (offenders.length > 0) {
        net.failed.push({ phase: 'contract', status: r.status(), method, url, pascalCaseKeys: offenders })
      }
    }

    if (r.status() < 400 && method === 'GET') return
    const rec = { phase: 'res', status: r.status(), method, url }
    try { rec.body = (await r.text()).slice(0, 700) } catch (_) { rec.body = '(unreadable)' }
    if (r.status() >= 400) net.failed.push(rec)
    net.calls.push(rec)
  })

  await page.goto(BASE + '/login')
  await page.fill('input[placeholder="請輸入電子郵件"]', LOGIN_EMAIL)
  await page.fill('input[placeholder="請輸入密碼"]', LOGIN_PASSWORD)
  await page.getByRole('button', { name: '登入系統' }).click()
  await page.waitForURL(u => !String(u).includes('/login'), { timeout: 20000 })
  await page.waitForTimeout(1200)
  return { browser, ctx, page, net }
}

function dialog(page) { return page.locator('.el-dialog:visible, .el-drawer:visible').last() }

function item(page, label) {
  return dialog(page).locator('.el-form-item')
    .filter({ has: page.locator(`.el-form-item__label:text-is("${label}")`) }).first()
}

async function dumpForm(page) {
  return await page.evaluate(() => {
    const visible = el => el && el.offsetParent !== null
    const box = [...document.querySelectorAll('.el-dialog, .el-drawer')].filter(visible).pop()
    if (!box) return { found: false }
    return {
      found: true,
      title: box.querySelector('.el-dialog__title, .el-drawer__title')?.innerText || '',
      items: [...box.querySelectorAll('.el-form-item')].map(fi => ({
        label: fi.querySelector('.el-form-item__label')?.innerText.trim() || '',
        required: fi.classList.contains('is-required'),
        value: [...fi.querySelectorAll('input,textarea')].map(c => c.value).filter(Boolean).join(' | '),
        selected: [...fi.querySelectorAll('.el-select__selected-item, .el-tag')].map(s => s.innerText.trim()).filter(Boolean).join(' | ')
      })),
      footer: [...box.querySelectorAll('.el-dialog__footer button, .el-drawer__footer button')].map(b => b.innerText.trim())
    }
  })
}

async function fill(page, label, value) {
  await item(page, label).locator('input:not([readonly]), textarea').first().fill(String(value))
  await page.waitForTimeout(120)
}

async function pick(page, label, optionText) {
  await item(page, label).locator('.el-select').first().click()
  await page.waitForTimeout(400)
  await page.locator('.el-select-dropdown:visible').last()
    .locator('.el-select-dropdown__item', { hasText: optionText }).first().click()
  await page.waitForTimeout(250)
}

async function selectOptions(page, label) {
  await item(page, label).locator('.el-select').first().click()
  await page.waitForTimeout(500)
  const opts = await page.locator('.el-select-dropdown:visible').last()
    .locator('.el-select-dropdown__item').allInnerTexts()
  // Esc 會冒泡關掉整個 el-dialog，改再點一次觸發器收起下拉。
  await item(page, label).locator('.el-select').first().click()
  await page.waitForTimeout(250)
  return opts.map(s => s.trim())
}

async function radio(page, label, text) {
  await item(page, label).locator('.el-radio, .el-radio-button', { hasText: text }).first().click()
  await page.waitForTimeout(150)
}

async function pickDate(page, label, value) {
  const input = item(page, label).locator('input').first()
  await input.click()
  await page.waitForTimeout(300)
  await input.fill(String(value))
  await page.waitForTimeout(300)
  await page.keyboard.press('Enter')
  await page.waitForTimeout(400)
  // Esc 會冒泡關掉整個 el-dialog，改點標題列收起日期面板。
  await dialog(page).locator('.el-dialog__header, .el-drawer__header').first().click({ force: true }).catch(() => {})
  await page.waitForTimeout(300)
}

async function errors(page) {
  return await page.evaluate(() => {
    const visible = el => el && el.offsetParent !== null
    const box = [...document.querySelectorAll('.el-dialog, .el-drawer')].filter(visible).pop()
    if (!box) return []
    return [...box.querySelectorAll('.el-form-item__error')].map(e => e.innerText.trim())
  })
}

async function toasts(page) {
  return await page.evaluate(() =>
    [...document.querySelectorAll('.el-message, .el-notification')]
      .map(m => m.innerText.replace(/\s+/g, ' ').trim()).filter(Boolean))
}

async function clearToasts(page) {
  // Element Plus owns the toast nodes; removing them breaks Vue's renderer and kills later interactions.
  await page.waitForFunction(
    () => document.querySelectorAll('.el-message, .el-notification').length === 0,
    null, { timeout: 6000 }
  ).catch(() => {})
}

async function submit(page, net, name = '儲存') {
  const before = net.calls.length
  await dialog(page).locator('button', { hasText: name }).last().click()
  await page.waitForTimeout(2600)
  const result = {
    calls: net.calls.slice(before),
    errors: await errors(page),
    toasts: await toasts(page),
    dialogOpen: (await page.locator('.el-dialog:visible, .el-drawer:visible').count()) > 0
  }
  await clearToasts(page)
  return result
}

// Element Plus 的 confirm 按鈕文案各頁不同（確定／刪除／確認），一律點最後一顆按鈕。
async function confirmBox(page, net) {
  const before = net ? net.calls.length : 0
  const box = page.locator('.el-message-box:visible').last()
  if (await box.count() === 0) return { found: false }
  const text = (await box.innerText()).replace(/\s+/g, ' ').trim()
  await box.locator('.el-message-box__btns button').last().click()
  await page.waitForTimeout(1600)
  const out = { found: true, text, toasts: await toasts(page) }
  if (net) out.calls = net.calls.slice(before)
  await clearToasts(page)
  return out
}

async function closeDialog(page) {
  for (let i = 0; i < 4; i++) {
    if ((await page.locator('.el-dialog:visible, .el-drawer:visible').count()) === 0) break
    const box = page.locator('.el-dialog:visible, .el-drawer:visible').last()
    const cancel = box.locator('button', { hasText: /^(取消|關閉)$/ }).last()
    if (await cancel.count() > 0) await cancel.click({ force: true }).catch(() => {})
    else await box.locator('.el-dialog__headerbtn, .el-drawer__close-btn').first().click({ force: true }).catch(() => {})
    await page.waitForTimeout(500)
  }
  await clearToasts(page)
}

async function openCreate(page, buttonName) {
  await closeDialog(page)
  await page.getByRole('button', { name: buttonName, exact: false }).first().click()
  await page.waitForTimeout(900)
}

async function goto(page, route, wait = 1800) {
  await closeDialog(page)
  await page.goto(BASE + route)
  await page.waitForTimeout(wait)
}

const DOWNLOAD_DIR = path.join(__dirname, 'downloads')

async function download(page, clickTarget) {
  require('node:fs').mkdirSync(DOWNLOAD_DIR, { recursive: true })
  const [dl] = await Promise.all([
    page.waitForEvent('download', { timeout: 30000 }),
    typeof clickTarget === 'function' ? clickTarget() : page.getByRole('button', { name: clickTarget, exact: false }).first().click()
  ])
  const name = dl.suggestedFilename()
  const file = path.join(DOWNLOAD_DIR, name)
  await dl.saveAs(file)
  const size = require('node:fs').statSync(file).size
  return { file, name, size }
}

async function upload(page, selector, filePath) {
  await page.setInputFiles(selector, filePath)
  await page.waitForTimeout(1500)
}

async function tableRows(page) {
  return await page.evaluate(() => {
    const body = document.querySelector('.el-table__body-wrapper .el-table__body')
      || document.querySelector('.el-table__body')
      || document.querySelector('table')
    if (!body) return []
    return [...body.querySelectorAll('tbody tr')]
      .map(tr => [...tr.querySelectorAll('td')].map(td => td.innerText.replace(/\s+/g, ' ').trim()))
      .filter(cells => cells.length > 0)
  })
}

function rowByText(page, text) {
  const exact = page.locator('.el-table__body tr')
    .filter({ has: page.locator(`td :text-is("${text}")`) }).first()
  return { exact, loose: page.locator('.el-table__body tr', { hasText: text }).first() }
}

async function resolveRow(page, text) {
  const { exact, loose } = rowByText(page, text)
  // 整格完全相符可避免「甲」同時命中「甲改」，命中不到才退回子字串比對。
  return (await exact.count()) > 0 ? exact : loose
}

async function rowAction(page, rowText, actionText) {
  const row = await resolveRow(page, rowText)
  await row.locator('button', { hasText: actionText }).first().click()
  await page.waitForTimeout(900)
}

module.exports = {
  BASE, TAG, launch, dialog, item, dumpForm, fill, pick, selectOptions, radio, pickDate,
  errors, toasts, clearToasts, submit, confirmBox, closeDialog, openCreate, goto,
  tableRows, rowByText, resolveRow, rowAction, download, upload, DOWNLOAD_DIR
}
