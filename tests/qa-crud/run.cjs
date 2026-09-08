const fs = require('node:fs')
const path = require('node:path')
const L = require('./lib.cjs')

const SUITE_DIR = path.join(__dirname, 'suites')

function listSuites() {
  return fs.readdirSync(SUITE_DIR).filter(f => f.endsWith('.cjs')).sort()
}

function selectSuites(args) {
  const all = listSuites()
  if (args.length === 0) return all
  return all.filter(f => args.some(a => f.includes(a)))
}

async function main() {
  const args = process.argv.slice(2).filter(a => !a.startsWith('-'))
  const files = selectSuites(args)
  if (files.length === 0) {
    console.error('no matching suite. available:\n  ' + listSuites().join('\n  '))
    process.exit(2)
  }

  const { browser, page, net } = await L.launch()
  const failedAssertions = []
  const report = { baseUrl: L.BASE, startedAt: new Date().toISOString(), suites: [] }

  for (const file of files) {
    const suite = require(path.join(SUITE_DIR, file))
    const steps = []
    const record = (name, data) => steps.push({ step: name, ...data })
    // 單一步驟失敗不應讓整個 suite 中斷，其餘檢查仍要跑完。
    const step = async (name, fn) => {
      try {
        record(name, await fn())
      } catch (e) {
        record(name, { stepError: String(e).split('\n')[0].slice(0, 300) })
        await L.closeDialog(page).catch(() => {})
      }
    }
    // 斷言結果記進 steps，讓報告可以直接看出哪一條回歸鎖破了。
    const expect = (name, ok, detail) => {
      record(name, { assertion: ok ? 'PASS' : 'FAIL', detail })
      if (!ok) failedAssertions.push({ suite: file, name, detail })
    }
    const failedBefore = net.failed.length
    let error = null
    try {
      await suite.run({ page, net, record, step, expect, L })
    } catch (e) {
      error = String(e).split('\n').slice(0, 3).join(' ').slice(0, 400)
    }
    report.suites.push({
      file,
      name: suite.name || file,
      error,
      steps,
      httpErrors: net.failed.slice(failedBefore)
    })
    await L.closeDialog(page).catch(() => {})
  }

  report.finishedAt = new Date().toISOString()
  report.failedAssertions = failedAssertions
  report.consoleErrors = net.console
  report.pageErrors = net.pageerror
  await browser.close()

  const outDir = path.join(__dirname, 'reports')
  fs.mkdirSync(outDir, { recursive: true })
  const outFile = path.join(outDir, `report-${Date.now()}.json`)
  fs.writeFileSync(outFile, JSON.stringify(report, null, 1), 'utf8')
  console.log(JSON.stringify(report, null, 1))
  console.error('report written to ' + outFile)
  if (failedAssertions.length > 0) {
    console.error('FAILED ASSERTIONS:')
    for (const a of failedAssertions) console.error(`  ${a.suite} :: ${a.name} :: ${JSON.stringify(a.detail)}`)
    process.exitCode = 1
  }
}

main().catch(e => { console.error('FATAL', e); process.exit(1) })
