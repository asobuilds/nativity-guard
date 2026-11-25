import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'
import { ApiError } from './apiClient'

export const QUERY_ERROR_EVENT = 'cs:query-error'

export interface QueryErrorDetail {
  source: 'query' | 'mutation'
  message: string
  error: unknown
  status?: number
  isOffline: boolean
  queryKey?: readonly unknown[]
  mutationKey?: readonly unknown[]
}

export type QueryErrorListener = (detail: QueryErrorDetail) => void

const errorListeners = new Set<QueryErrorListener>()

/** Subscribe to unhandled React Query query/mutation errors. */
export function subscribeQueryErrors(listener: QueryErrorListener): () => void {
  errorListeners.add(listener)
  return () => {
    errorListeners.delete(listener)
  }
}

/** True when the failure looks like a connectivity/offline error rather than an HTTP response. */
export function isOfflineError(error: unknown): boolean {
  if (ApiError.isNetwork(error)) return true
  if (error instanceof TypeError && /fetch|network|offline/i.test(error.message)) return true
  if (typeof navigator !== 'undefined' && navigator.onLine === false && !(error instanceof ApiError)) {
    return true
  }
  return false
}

/** Extract a user-facing error message from an unknown thrown value. */
export function extractErrorMessage(
  error: unknown,
  fallback = 'Something went wrong. Please try again.',
): string {
  if (isOfflineError(error)) {
    return 'You appear to be offline. Check your connection and try again.'
  }
  if (error instanceof ApiError && error.message.trim()) {
    return error.message
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  if (typeof error === 'string' && error.trim()) {
    return error
  }
  return fallback
}

export interface GlobalErrorFilterOptions {
  meta?: Record<string, unknown>
  hasLocalHandler?: boolean
  isOfflineFirst?: boolean
}

/**
 * Decide whether a query or mutation failure should be surfaced globally.
 *
 * - Explicit opt-out via `meta: { silent: true }` or `meta: { suppressGlobalError: true }`.
 * - Mutations with a local `onError` callback already handle their own feedback.
 * - 401 responses are handled centrally via `UNAUTHORIZED_EVENT` in `apiClient.ts`.
 * - `offlineFirst` public queries (e.g. `usePublicBlueprint`, `usePublicLeaderboard`)
 *   must not emit error noise when offline.
 */
export function shouldSurfaceGlobalError(
  error: unknown,
  options: GlobalErrorFilterOptions = {},
): boolean {
  const { meta, hasLocalHandler = false, isOfflineFirst = false } = options
  if (meta?.silent === true || meta?.suppressGlobalError === true) return false
  if (hasLocalHandler) return false
  if (error instanceof ApiError && error.status === 401) return false
  if (isOfflineFirst && isOfflineError(error)) return false
  return true
}

function emitQueryError(detail: QueryErrorDetail): void {
  for (const listener of errorListeners) {
    try {
      listener(detail)
    } catch {
      /* isolate listener failures */
    }
  }
  if (typeof window !== 'undefined' && typeof CustomEvent !== 'undefined') {
    window.dispatchEvent(new CustomEvent<QueryErrorDetail>(QUERY_ERROR_EVENT, { detail }))
  }
}

function buildErrorDetail(
  source: 'query' | 'mutation',
  error: unknown,
  keys: { queryKey?: readonly unknown[]; mutationKey?: readonly unknown[] },
): QueryErrorDetail {
  return {
    source,
    message: extractErrorMessage(error),
    error,
    status: error instanceof ApiError ? error.status : undefined,
    isOffline: isOfflineError(error),
    ...keys,
  }
}

/**
 * Shared React Query client factory.
 *
 * Retries: never retry auth/permission/validation failures (4xx) or explicit
 * offline failures; retry twice on network/5xx when online.
 *
 * Cache window: five minutes is the sweet spot for a safety app — data
 * changes on the order of minutes, not seconds, and every refetch costs a
 * round-trip on rural connections. `gcTime` keeps a page's data alive for
 * 30 min after the user navigates away so returning is instant.
 */
export function createAppQueryClient(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error, query) => {
        if (
          !shouldSurfaceGlobalError(error, {
            meta: query.meta as Record<string, unknown> | undefined,
            isOfflineFirst: query.options.networkMode === 'offlineFirst',
          })
        ) {
          return
        }
        emitQueryError(buildErrorDetail('query', error, { queryKey: query.queryKey }))
      },
    }),
    mutationCache: new MutationCache({
      onError: (error, _variables, _context, mutation) => {
        if (
          !shouldSurfaceGlobalError(error, {
            meta: mutation.meta as Record<string, unknown> | undefined,
            hasLocalHandler: typeof mutation.options.onError === 'function',
            isOfflineFirst: mutation.options.networkMode === 'offlineFirst',
          })
        ) {
          return
        }
        emitQueryError(
          buildErrorDetail('mutation', error, { mutationKey: mutation.options.mutationKey }),
        )
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 5 * 60_000, // 5 minutes — no refetch on navigation within this window
        gcTime: 30 * 60_000, // 30 minutes — keep cached data available after nav away
        refetchOnWindowFocus: false,
        refetchOnReconnect: true, // but DO refresh after coming back from offline
        retry: (failureCount, error) => {
          if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
            return false
          }
          if (typeof navigator !== 'undefined' && navigator.onLine === false && isOfflineError(error)) {
            return false
          }
          return failureCount < 2
        },
      },
      mutations: {
        retry: 0,
      },
    },
  })
}

export const queryClient = createAppQueryClient()
