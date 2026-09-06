import assert from 'node:assert/strict'
import test from 'node:test'

import { createPaginationMeta, unwrapData, unwrapPaged } from '../../src/api/envelope.ts'

test('unwrapData extracts the shared API envelope without changing raw values', () => {
  assert.deepEqual(unwrapData<{ id: string }>({ data: { id: '1' } }), { id: '1' })
  assert.deepEqual(unwrapData<string[]>(['a']), ['a'])
})

test('unwrapPaged keeps server metadata and falls back only when it is absent', () => {
  const fallback = createPaginationMeta(2, 10)
  assert.deepEqual(
    unwrapPaged({ data: [{ id: '1' }], meta: { page: 1, pageSize: 1, total: 3, totalPages: 3 } }, fallback),
    { data: [{ id: '1' }], meta: { page: 1, pageSize: 1, total: 3, totalPages: 3 } }
  )
  assert.deepEqual(unwrapPaged([{ id: '2' }], fallback), { data: [{ id: '2' }], meta: fallback })
  assert.deepEqual(unwrapPaged({ data: 'invalid' }, fallback), { data: [], meta: fallback })
})
