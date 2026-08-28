#!/usr/bin/env node

/**
 * 智慧比對 Git 改動並執行對應前端功能之 E2E 測試
 */
import { execSync } from 'child_process'
import path from 'path'

const FEATURE_SPEC_MAP = [
  {
    spec: 'tests/e2e/01-auth.spec.ts',
    name: '認證與權限',
    patterns: [/views\/auth/, /stores\/auth\.ts/, /router\/guards\.ts/]
  },
  {
    spec: 'tests/e2e/02-dashboard.spec.ts',
    name: '總覽儀表板',
    patterns: [/views\/dashboard/, /api\/dashboard\.ts/]
  },
  {
    spec: 'tests/e2e/03-cases.spec.ts',
    name: '個案與排班',
    patterns: [/views\/cases/, /api\/cases\.ts/, /components\/MaskedId\.vue/, /components\/ImportPreviewDialog\.vue/]
  },
  {
    spec: 'tests/e2e/04-masters.spec.ts',
    name: '基礎主檔管理',
    patterns: [/views\/masters/, /api\/masters\.ts/]
  },
  {
    spec: 'tests/e2e/05-forms.spec.ts',
    name: '表單同步與對應',
    patterns: [/views\/forms/, /api\/forms\.ts/]
  },
  {
    spec: 'tests/e2e/06-rides.spec.ts',
    name: '搭乘月曆與更正',
    patterns: [
      /views\/rides\/RideCalendarView\.vue/,
      /views\/rides\/RideCorrectionDrawer\.vue/,
      /views\/rides\/RideManualEntryDialog\.vue/,
      /api\/rides\.ts/
    ]
  },
  {
    spec: 'tests/e2e/07-ride-issues.spec.ts',
    name: '異常集中與未回報',
    patterns: [/views\/rides\/RideIssuesView\.vue/, /views\/rides\/MissingRidesView\.vue/]
  },
  {
    spec: 'tests/e2e/08-reports.spec.ts',
    name: '營運報表',
    patterns: [/views\/reports/, /api\/reports\.ts/]
  },
  {
    spec: 'tests/e2e/09-operations.spec.ts',
    name: '保養與出勤油資',
    patterns: [/views\/vehicles\/MaintenanceView\.vue/, /views\/attendance/, /api\/maintenance\.ts/, /api\/attendance\.ts/]
  },
  {
    spec: 'tests/e2e/10-settings-and-audit.spec.ts',
    name: '稽核與系統設定',
    patterns: [/views\/audit/, /views\/settings/, /api\/audit\.ts/, /api\/users\.ts/, /api\/roles\.ts/, /api\/notifications\.ts/]
  },
  {
    spec: 'tests/e2e/11-exports.spec.ts',
    name: '政府申報匯出',
    patterns: [/views\/exports/, /components\/PrecheckResult\.vue/, /api\/exports\.ts/]
  }
]

function getChangedFiles() {
  try {
    const statusOut = execSync('git status --porcelain', { encoding: 'utf-8' })
    const files = statusOut
      .split('\n')
      .map((line) => line.trim().slice(3))
      .filter(Boolean)
    return files
  } catch (err) {
    return []
  }
}

function resolveSpecsToRun(changedFiles) {
  const matchedSpecs = new Set()
  const matchedFeatures = []

  // 若使用者帶入參數，直接比對特定名稱
  const arg = process.argv[2]
  if (arg) {
    const matched = FEATURE_SPEC_MAP.find((m) => m.spec.includes(arg) || m.name.includes(arg))
    if (matched) {
      return { specs: [matched.spec], names: [matched.name] }
    }
  }

  for (const file of changedFiles) {
    for (const item of FEATURE_SPEC_MAP) {
      if (item.patterns.some((regex) => regex.test(file))) {
        if (!matchedSpecs.has(item.spec)) {
          matchedSpecs.add(item.spec)
          matchedFeatures.push(item.name)
        }
      }
    }
  }

  return {
    specs: Array.from(matchedSpecs),
    names: matchedFeatures
  }
}

function main() {
  const changedFiles = getChangedFiles()
  console.log('🔍 檢查目前 Git 工作區改動檔案...')

  const { specs, names } = resolveSpecsToRun(changedFiles)

  if (specs.length === 0) {
    console.log('ℹ️ 未偵測到特定功能頁面改動或改動為全域設定，即將執行全量 E2E 測試...')
    execSync('npx playwright test', { stdio: 'inherit' })
    return
  }

  console.log(`🎯 偵測到受影響功能模組：[ ${names.join(', ')} ]`)
  console.log(`🚀 開始執行對應 E2E 測試：\n  - ${specs.join('\n  - ')}\n`)

  const cmd = `npx playwright test ${specs.join(' ')}`
  execSync(cmd, { stdio: 'inherit' })
}

main()

