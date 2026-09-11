import type { Paged, PaginationMeta } from '@/types/api'

type ApiRecord = Record<string, unknown>

function isRecord(value: unknown): value is ApiRecord {
  return typeof value === 'object' && value !== null
}

function isPaginationMeta(value: unknown): value is PaginationMeta {
  if (!isRecord(value)) return false
  return (
    typeof value.page === 'number' &&
    typeof value.pageSize === 'number' &&
    typeof value.total === 'number' &&
    typeof value.totalPages === 'number'
  )
}

// API module 的唯一資料邊界：攔截器保留完整 envelope，呼叫端在此明確取出 data。
export function unwrapData<T>(response: unknown): T {
  if (isRecord(response) && 'data' in response) {
    return response.data as T
  }
  return response as T
}

export interface PendingRelinkedMeta {
  pendingRelinked?: number
}

export function unwrapDataWithMeta<T>(response: unknown): { data: T; meta?: PendingRelinkedMeta } {
  const data = unwrapData<T>(response)
  const meta = isRecord(response) && isRecord(response.meta) ? (response.meta as PendingRelinkedMeta) : undefined
  return { data, meta }
}

export function createPaginationMeta(page = 1, pageSize = 20, total = 0): PaginationMeta {
  return {
    page,
    pageSize,
    total,
    totalPages: total === 0 ? 0 : Math.ceil(total / pageSize)
  }
}

export function unwrapPaged<T>(response: unknown, fallback: PaginationMeta): Paged<T> {
  if (isRecord(response) && Array.isArray(response.data)) {
    return {
      data: response.data as T[],
      meta: isPaginationMeta(response.meta) ? response.meta : fallback
    }
  }

  const data = unwrapData<unknown>(response)
  if (Array.isArray(data)) {
    return { data: data as T[], meta: fallback }
  }

  if (isRecord(data) && Array.isArray(data.data)) {
    return {
      data: data.data as T[],
      meta: isPaginationMeta(data.meta) ? data.meta : fallback
    }
  }

  return { data: [], meta: fallback }
}
