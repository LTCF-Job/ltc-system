import { ref, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export interface ListQueryOptions<TFilter extends Record<string, any>> {
  defaultFilters: TFilter
  defaultPageSize?: number
  onFetch: () => void | Promise<void>
}

// 支援清單頁查詢條件與網址參數雙向同步
export function useListQuery<TFilter extends Record<string, any>>(options: ListQueryOptions<TFilter>) {
  const route = useRoute()
  const router = useRouter()

  const page = ref<number>(Number(route.query.page) || 1)
  const pageSize = ref<number>(Number(route.query.pageSize) || options.defaultPageSize || 20)
  const total = ref<number>(0)
  const loading = ref<boolean>(false)

  const filters = reactive<TFilter>({
    ...options.defaultFilters,
    ...extractFiltersFromQuery(route.query, options.defaultFilters)
  }) as TFilter

  function extractFiltersFromQuery(query: Record<string, any>, defaults: TFilter): Partial<TFilter> {
    const res: any = {}
    for (const key of Object.keys(defaults)) {
      if (query[key] !== undefined && query[key] !== '') {
        res[key] = query[key]
      }
    }
    return res
  }

  function syncToQuery() {
    const query: Record<string, any> = {
      page: page.value > 1 ? page.value : undefined,
      pageSize: pageSize.value !== (options.defaultPageSize || 20) ? pageSize.value : undefined
    }

    for (const [key, value] of Object.entries(filters)) {
      if (value !== undefined && value !== '' && value !== null) {
        query[key] = value
      }
    }

    router.replace({ query })
  }

  async function executeFetch() {
    loading.value = true
    try {
      await options.onFetch()
    } finally {
      loading.value = false
    }
  }

  function handlePageChange(newPage: number) {
    page.value = newPage
    syncToQuery()
    executeFetch()
  }

  function handleSizeChange(newSize: number) {
    pageSize.value = newSize
    page.value = 1
    syncToQuery()
    executeFetch()
  }

  function handleSearch() {
    page.value = 1
    syncToQuery()
    executeFetch()
  }

  function handleReset() {
    Object.assign(filters, options.defaultFilters)
    page.value = 1
    syncToQuery()
    executeFetch()
  }

  return {
    page,
    pageSize,
    total,
    loading,
    filters,
    handlePageChange,
    handleSizeChange,
    handleSearch,
    handleReset,
    executeFetch
  }
}
