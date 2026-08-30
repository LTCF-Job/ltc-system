// 匯報欄位可對應的排班趟次。四趟個案由後端在寫入時把第 1／2 趟展開為 1、3 與 2、4，
// 因此此處四個選項與 schedule_legs 的 leg_seq 一致。
export const LEG_SEQ_OPTIONS = [
  { value: 1, label: '第 1 趟 (去程)' },
  { value: 2, label: '第 2 趟 (回程)' },
  { value: 3, label: '第 3 趟 (去程)' },
  { value: 4, label: '第 4 趟 (回程)' }
] as const
