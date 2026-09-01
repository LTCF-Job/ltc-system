import * as mockData from './mockData'
import { mockHolidays } from '../handlers/holidays'
import { mockRideOverrides } from '../handlers/rides'

const STORAGE_KEY = 'ltc_demo_data'
// mockData.ts 內容有結構性變動（新增/調整資料集鍵值、資料形狀）時遞增，讓舊版持久化快照失效，避免蓋掉新版展示資料
const DEMO_DATA_VERSION = 6

// mockData.ts 匯出的資料集（陣列與物件皆有）＋ handler 內獨立維護的展示狀態，共同構成展示模式的可持久化資料
const MOCK_DATA_KEYS = Object.keys(mockData) as (keyof typeof mockData)[]
const stores: Record<string, unknown> = {
  ...Object.fromEntries(MOCK_DATA_KEYS.map((key) => [key, mockData[key]])),
  mockHolidays,
  mockRideOverrides
}

// 模組載入當下（尚未有任何請求 mutate 過）拍下的快照，供「登出重置」還原成最初的展示資料
const initialSnapshot = snapshot()

function deepClone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value))
}

function snapshot(): Record<string, unknown> {
  return Object.fromEntries(Object.entries(stores).map(([key, value]) => [key, deepClone(value)]))
}

// 保留原本的陣列／物件參照，只替換內容，讓既有 handler 的 push／splice／Object.assign 寫法不需改動
function replaceInPlace(target: unknown, source: unknown) {
  if (Array.isArray(target) && Array.isArray(source)) {
    target.length = 0
    target.push(...source)
  } else if (target && typeof target === 'object' && source && typeof source === 'object') {
    for (const key of Object.keys(target)) delete (target as Record<string, unknown>)[key]
    Object.assign(target, source)
  }
}

function applySnapshot(snap: Record<string, unknown>) {
  for (const key of Object.keys(stores)) replaceInPlace(stores[key], snap[key])
}

// 展示模式啟動時呼叫：還原上次寫入後持久化的展示資料，讓重新整理頁面後仍看得到
export function restoreDemoData() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      // 快照版本與目前 mockData 結構不符（改版新增的資料集鍵值/形狀）時視為過期，改用最新展示資料
      if (parsed.version === DEMO_DATA_VERSION) {
        applySnapshot(parsed.data)
        return
      }
    }
  } catch {
    // localStorage 內容毀損時退回初始展示資料，避免整站因壞資料無法啟動
  }
  localStorage.removeItem(STORAGE_KEY)
  applySnapshot(initialSnapshot)
}

// 展示模式下每次寫入類請求完成後呼叫：落地目前狀態，讓重新整理不遺失
export function persistDemoData() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ version: DEMO_DATA_VERSION, data: snapshot() }))
  } catch {
    // storage 滿版或被封鎖時展示模式仍可運作，只是重新整理會遺失本次寫入
  }
}

// 登出時呼叫：清除持久化紀錄並將記憶體資料還原為最初的展示資料集
export function resetDemoData() {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore：清不掉持久化紀錄不影響記憶體立即還原
  }
  applySnapshot(initialSnapshot)
}
