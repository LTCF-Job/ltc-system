import { createClient } from '@supabase/supabase-js'
import { shouldUseSupabaseAuth } from './authMode'

const supabaseUrl = (import.meta.env.VITE_SUPABASE_URL as string | undefined)?.trim()
const supabaseAnonKey = (import.meta.env.VITE_SUPABASE_ANON_KEY as string | undefined)?.trim()
const hasSupabaseCredentials = Boolean(supabaseUrl && supabaseAnonKey)

// local 一律走本機 mock 登入，避免被 .env.local 殘留的 Supabase 設定切換到雲端 session。
// 非 local 環境只有在兩個必要設定都存在時才建立 Supabase client。
export const supabase = shouldUseSupabaseAuth({
  isDev: import.meta.env.DEV,
  appEnv: import.meta.env.VITE_APP_ENV,
  hasSupabaseCredentials
})
  ? createClient(supabaseUrl!, supabaseAnonKey!)
  : null
