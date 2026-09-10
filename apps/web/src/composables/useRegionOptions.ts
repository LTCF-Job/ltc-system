import { ref } from 'vue'
import { listAllSites } from '@/api/masters'

// 區域選項（sites.region）
//
// 區域已無主檔（regions 表於 migration 000048 移除），值只存在於 sites.region 這個自由
// 文字欄位，因此選項一律由據點主檔即時彙整而來。比對的是個案關聯的據點（含停用），
// 不能只抓啟用據點，否則個案掛在停用據點時整個區域會從選單消失。
export function useRegionOptions() {
  const regionOptions = ref<string[]>([])
  // 每個區域底下的據點數。區域是自由文字，錯字或全半形差異會讓同一個區域裂成兩個選項，
  // 標上據點數讓這種異常在選單上一眼看得出來。
  const siteCountByRegion = ref<Record<string, number>>({})
  const loadingRegions = ref(false)

  async function refreshRegionOptions() {
    loadingRegions.value = true
    try {
      const allSites = await listAllSites()
      const counts: Record<string, number> = {}
      for (const site of allSites) {
        if (!site.region) continue
        counts[site.region] = (counts[site.region] ?? 0) + 1
      }
      siteCountByRegion.value = counts
      regionOptions.value = Object.keys(counts).sort((a, b) => a.localeCompare(b, 'zh-Hant'))
    } finally {
      loadingRegions.value = false
    }
  }

  return { regionOptions, siteCountByRegion, loadingRegions, refreshRegionOptions }
}
