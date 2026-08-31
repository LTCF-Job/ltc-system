/**
 * useGooglePicker composable
 * 整合 Google Identity Services (GIS) 與 Google Picker API
 * 支援彈出 Google 原生帳號登入授權視窗與雲端硬碟試算表挑選器
 */

import { ref, onMounted } from 'vue'

export interface GooglePickedFile {
  id: string
  name: string
  url: string
  mimeType: string
  accessToken: string
}

export interface GooglePickerOptions {
  clientId: string
  apiKey: string
  appId?: string
  scopes?: string[]
}

declare global {
  interface Window {
    google?: any
    gapi?: any
  }
}

export function useGooglePicker() {
  const isLoaded = ref(false)
  const isAuthorizing = ref(false)
  const isPickerOpen = ref(false)
  const accessToken = ref<string | null>(null)
  const error = ref<string | null>(null)

  // 1. 動態載入 Google GIS (Identity Services) SDK
  function loadGsiScript(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (window.google?.accounts?.oauth2) {
        return resolve()
      }
      const existing = document.querySelector('script[src="https://accounts.google.com/gsi/client"]')
      if (existing) {
        existing.addEventListener('load', () => resolve())
        existing.addEventListener('error', (e) => reject(e))
        return
      }
      const script = document.createElement('script')
      script.src = 'https://accounts.google.com/gsi/client'
      script.async = true
      script.defer = true
      script.onload = () => resolve()
      script.onerror = (e) => reject(e)
      document.head.appendChild(script)
    })
  }

  // 2. 動態載入 Google GAPI (Picker) SDK
  function loadGapiScript(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (window.gapi?.picker) {
        return resolve()
      }
      const existing = document.querySelector('script[src="https://apis.google.com/js/api.js"]')
      if (existing) {
        if (window.gapi) {
          window.gapi.load('picker', { callback: () => resolve() })
        } else {
          existing.addEventListener('load', () => {
            window.gapi.load('picker', { callback: () => resolve() })
          })
        }
        return
      }
      const script = document.createElement('script')
      script.src = 'https://apis.google.com/js/api.js'
      script.async = true
      script.defer = true
      script.onload = () => {
        if (window.gapi) {
          window.gapi.load('picker', { callback: () => resolve() })
        } else {
          resolve()
        }
      }
      script.onerror = (e) => reject(e)
      document.head.appendChild(script)
    })
  }

  // 確保 Google SDK 皆已載入
  async function ensureSdkLoaded(): Promise<void> {
    if (isLoaded.value) return
    await Promise.all([loadGsiScript(), loadGapiScript()])
    isLoaded.value = true
  }

  // 3. 取得 Google OAuth 2.0 Access Token（彈出 Google 登入授權視窗）
  function requestAccessToken(clientId: string, scopes: string[]): Promise<string> {
    return new Promise((resolve, reject) => {
      if (!window.google?.accounts?.oauth2) {
        return reject(new Error('Google Identity Services SDK 尚未載入完成'))
      }

      let resolved = false
      const client = window.google.accounts.oauth2.initTokenClient({
        client_id: clientId,
        scope: scopes.join(' '),
        callback: (response: any) => {
          if (resolved) return
          resolved = true
          if (response.error) {
            console.error('[Google OAuth Error]', response)
            return reject(new Error(`Google 授權失敗: ${response.error_description || response.error}`))
          }
          if (response.access_token) {
            accessToken.value = response.access_token
            resolve(response.access_token)
          } else {
            reject(new Error('未取得 Google Access Token'))
          }
        },
        error_callback: (err: any) => {
          if (resolved) return
          resolved = true
          console.error('[Google OAuth Error Callback]', err)
          reject(new Error(`Google 登入視窗開啟失敗: ${err.message || JSON.stringify(err)}`))
        }
      })

      // 觸發彈出 Google 登入與授權視窗
      client.requestAccessToken({ prompt: 'consent' })
    })
  }

  // 4. 開啟 Google Picker 原生檔案選擇視窗
  async function openPicker(options: GooglePickerOptions): Promise<GooglePickedFile | null> {
    error.value = null
    isAuthorizing.value = true

    const scopes = options.scopes || [
      'https://www.googleapis.com/auth/drive.readonly',
      'https://www.googleapis.com/auth/spreadsheets.readonly'
    ]

    try {
      await ensureSdkLoaded()

      // 取得或重複使用 Access Token
      let token = accessToken.value
      if (!token) {
        token = await requestAccessToken(options.clientId, scopes)
      }
      isAuthorizing.value = false

      if (!token) {
        throw new Error('未取得有效的 Google 授權 Token')
      }

      if (!window.google?.picker) {
        await new Promise((resolve) => {
          window.gapi.load('picker', { callback: () => resolve(true) })
        })
      }

      if (!window.google?.picker) {
        throw new Error('Google Picker API 未成功載入')
      }

      isPickerOpen.value = true

      return new Promise<GooglePickedFile | null>((resolve, reject) => {
        try {
          const view = new window.google.picker.DocsView(window.google.picker.ViewId.SPREADSHEETS)
          view.setMimeTypes('application/vnd.google-apps.spreadsheet')
          view.setMode(window.google.picker.DocsViewMode.LIST)

          const origin = window.location.protocol + '//' + window.location.host
          const builder = new window.google.picker.PickerBuilder()
            .addView(view)
            .setOAuthToken(token)
            .setDeveloperKey(options.apiKey)
            .setOrigin(origin)
            .setTitle('選擇要關聯的 Google 試算表')
            .setCallback((data: any) => {
              if (data.action === window.google.picker.Action.PICKED) {
                const doc = data.docs[0]
                isPickerOpen.value = false
                resolve({
                  id: doc.id,
                  name: doc.name,
                  url: doc.url || `https://docs.google.com/spreadsheets/d/${doc.id}/edit`,
                  mimeType: doc.mimeType,
                  accessToken: token!
                })
              } else if (data.action === window.google.picker.Action.CANCEL) {
                isPickerOpen.value = false
                resolve(null)
              }
            })

          if (options.appId) {
            builder.setAppId(options.appId)
          }

          const picker = builder.build()
          picker.setVisible(true)
        } catch (err: any) {
          isPickerOpen.value = false
          console.error('[Google Picker Builder Error]', err)
          reject(err)
        }
      })
    } catch (err: any) {
      isAuthorizing.value = false
      isPickerOpen.value = false
      error.value = err.message || '開啟 Google 選擇視窗失敗'
      throw err
    }
  }

  onMounted(() => {
    // 預先在背景載入 SDK，以確保使用者點選時能直接同步開啟 Google 登入彈跳視窗
    ensureSdkLoaded().catch((err) => {
      console.warn('[useGooglePicker] Preload SDK warning:', err)
    })
  })

  return {
    isLoaded,
    isAuthorizing,
    isPickerOpen,
    accessToken,
    error,
    openPicker
  }
}

