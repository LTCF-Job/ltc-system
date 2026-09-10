import { computed, ref, type Ref } from 'vue'
import { useRocMonth } from '@/composables/useRocMonth'

// 多月份選擇（民國）
//
// el-date-picker 的 months 型別給的是西元 `YYYY-MM` 陣列，後端要的是民國 5 碼（11507）。
// 轉換一律走 useRocMonth，與單月選擇共用同一套規則，避免兩邊各解析一次而分岔。
export function useRocMonthRange(initial: string[] = []) {
  const { toRocMonth, toRocPeriodYmList, formatRocMonthLabel } = useRocMonth()

  const selectedMonths: Ref<string[]> = ref([...initial])

  // 換算、去重與排序的規則放在 useRocMonth，與單月選擇共用同一套並受單元測試鎖定。
  const periodYms = computed(() => toRocPeriodYmList(selectedMonths.value))

  const rocMonthLabels = computed(() =>
    [...selectedMonths.value]
      .sort()
      .map((month) => formatRocMonthLabel(toRocMonth(month)))
  )

  const summary = computed(() => {
    if (selectedMonths.value.length === 0) return '尚未選擇月份'
    return `已選擇 ${periodYms.value.length} 個月：${rocMonthLabels.value.join('、')}`
  })

  return { selectedMonths, periodYms, rocMonthLabels, summary }
}
