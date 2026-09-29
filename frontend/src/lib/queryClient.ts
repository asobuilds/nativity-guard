import { QueryClient } from '@tanstack/react-query'
import { ApiError } from './apiClient'

/**
 * Shared React Query client.
 *
 * Retries: never retry auth/permission/validation failures; retry twice on
 * network or 5xx.
 *
 * Cache window: five minutes is the sweet spot for a safety app — data
 * changes on the order of minutes, not seconds, and every refetch costs a
 * round-trip on rural connections. `gcTime` keeps a page's data alive for
 * 30 min after the user navigates away so returning is instant.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60_000, // 5 minutes — no refetch on navigation within this window
      gcTime: 30 * 60_000,   // 30 minutes — keep cached data available after nav away
      refetchOnWindowFocus: false,
      refetchOnReconnect: true, // but DO refresh after coming back from offline
      retry: (failureCount, error) => {
        if (error instanceof ApiError) {
          if (error.status >= 400 && error.status < 500) return false
        }
        return failureCount < 2
      },
    },
    mutations: {
      retry: 0,
    },
  },
})