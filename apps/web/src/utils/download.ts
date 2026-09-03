/**
 * downloadBlob 將 API 回傳的檔案 Blob 交給瀏覽器下載。
 */
export function downloadBlob(blob: Blob, filename: string): void {
  const objectUrl = window.URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = filename
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()

  // 讓瀏覽器完成下載導覽後再釋放 URL，避免部分瀏覽器取得空檔案。
  window.setTimeout(() => {
    anchor.remove()
    window.URL.revokeObjectURL(objectUrl)
  }, 1000)
}
