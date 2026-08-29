// MSW 的 Service Worker 會攔截頁面發出的所有網路請求，不只是後端 API——Vite 開發伺服器
// 載入 .vue／.ts 模組、HMR 與靜態資源也會經過同一層。因此不能對所有未命中 handler 的請求
// 一律視為錯誤，只有打向後端 API（/api/v1/*）且沒有對應 mock 的請求才需要擋下，
// 避免展示模式帶著假憑證真的打到正式後端；其餘請求一律放行，否則整個頁面都載入不了。
export function onUnhandledRequest(request: Request, print: { warning: () => void; error: () => void }) {
  const { pathname } = new URL(request.url)
  if (pathname.startsWith('/api/v1/')) {
    print.error()
  }
}
