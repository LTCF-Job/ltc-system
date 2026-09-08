import { listAllRegions } from './masters'
import { REGION_LABELS } from '@/types/domain'
import type { RegionDTO } from '@/types/api'

export interface RegionOption {
  code: string
  name: string
}

let cache: Promise<RegionDTO[]> | null = null

function loadRegions(): Promise<RegionDTO[]> {
  if (!cache) {
    cache = listAllRegions().catch((err) => {
      // 失敗不留下壞掉的 promise，下次呼叫要能重試。
      cache = null
      throw err
    })
  }
  return cache
}

/** 讓下拉選項只列出啟用中的地區，依主檔排序權重排列。 */
export async function fetchRegionOptions(): Promise<RegionOption[]> {
  let regions: RegionDTO[]
  try {
    regions = await loadRegions()
  } catch {
    // 沒有地區主檔檢視權或主檔暫時讀不到時，退回保底字典，下拉不會整個空掉。
    return Object.entries(REGION_LABELS).map(([code, name]) => ({ code, name }))
  }
  // 自訂地區不在保底字典裡，沒補上就只會顯示 region_xxxxxxxx。
  for (const r of regions) nameByCode[r.code] = r.name
  return regions
    .filter((r) => r.status === 'active')
    .sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name))
    .map((r) => ({ code: r.code, name: r.name }))
}

// 稽核紀錄等頁面需要同步查表，因此以寫死的 REGION_LABELS 為保底字典。
const nameByCode: Record<string, string> = { ...REGION_LABELS }

/** 把 region code 翻成顯示名稱；查不到時原樣回傳。 */
export function regionLabel(code?: string | null): string {
  if (!code) return ''
  return nameByCode[code] || code
}

export async function primeRegionLabels(): Promise<void> {
  try {
    for (const r of await loadRegions()) {
      nameByCode[r.code] = r.name
    }
  } catch {
    // 保底字典已涵蓋 22 個縣市，載入失敗不影響既有顯示。
  }
}

export function resetRegionCache(): void {
  cache = null
}
