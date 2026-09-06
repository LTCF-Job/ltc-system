import assert from 'node:assert/strict'
import test from 'node:test'
import { shouldUseSupabaseAuth } from '../../src/lib/authMode.ts'

test('local environment ignores Supabase credentials and uses the local auth path', () => {
  assert.equal(
    shouldUseSupabaseAuth({
      isDev: false,
      appEnv: 'local',
      hasSupabaseCredentials: true
    }),
    false
  )
})

test('production environment uses Supabase when credentials are available', () => {
  assert.equal(
    shouldUseSupabaseAuth({
      isDev: false,
      appEnv: 'production',
      hasSupabaseCredentials: true
    }),
    true
  )
})
