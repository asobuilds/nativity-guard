import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface LiveNewsItem {
  id: string
  title: string
  summary: string
  url: string
  section: string
  publishedAt: string
  thumbnail?: string
  category: string
}

interface LiveNewsResponse {
  items: LiveNewsItem[]
  categories: string[]
}

/**
 * Live news from the backend, filtered to the caller's subscription when no
 * explicit `categories` are passed. The backend caches for 15 minutes, so
 * fast repeat fetches are free.
 */
export function useLiveNews(
  options: { categories?: string[]; limit?: number } = {},
) {
  const { categories, limit = 10 } = options
  const catKey = categories && categories.length > 0 ? categories.join(',') : 'from-subscription'

  return useQuery({
    queryKey: ['live-news', catKey, limit],
    queryFn: () => {
      const qs = new URLSearchParams()
      if (categories && categories.length > 0) qs.set('categories', categories.join(','))
      qs.set('limit', String(limit))
      return api.get<LiveNewsResponse>(`/news/live?${qs.toString()}`)
    },
    staleTime: 10 * 60_000,
    refetchOnWindowFocus: false,
    retry: 1,
  })
}