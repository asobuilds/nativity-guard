import { useQuery } from '@tanstack/react-query'
import { API_BASE } from '@/lib/apiClient'

export type PublicServiceAvailability = 'available' | 'unavailable' | 'unknown'
const REQUEST_TIMEOUT_MS = 8_000

/**
 * Resolve the backend's public /health URL from the same API origin configured
 * for normal application requests. Never infer service availability from the
 * frontend's own static index.html or from a fabricated status response.
 */
export function publicHealthUrl(apiBase: string, pageUrl: string): string {
  const url = new URL(apiBase, pageUrl)
  url.pathname = url.pathname.replace(/\/api\/v1\/?$/, '').replace(/\/$/, '') + '/health'
  url.search = ''
  url.hash = ''
  return url.toString()
}

/**
 * Browser-only, read-only reachability check. A non-2xx request (except server
 * errors) is ambiguous rather than evidence that the entire backend is down.
 * A 200 response is only treated as healthy if it contains a valid health
 * payload, including an affirmative database check.
 */
export async function probePublicService(
  url: string,
  fetcher: typeof fetch = fetch,
  timeoutMs = REQUEST_TIMEOUT_MS,
): Promise<PublicServiceAvailability> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), timeoutMs)
  try {
    const res = await fetcher(url, {
      method: 'GET',
      signal: controller.signal,
      headers: { Accept: 'application/json' },
      cache: 'no-store',
    })
    if (res.status >= 500) return 'unavailable'
    if (!res.ok) return 'unknown'
    const payload: unknown = await res.json()
    if (!payload || typeof payload !== 'object') return 'unknown'
    const record = payload as { status?: unknown; checks?: { database?: { ok?: unknown } } }
    if (record.status === 'down' || record.checks?.database?.ok === false) return 'unavailable'
    if (
      (record.status === 'ok' || record.status === 'degraded') &&
      record.checks?.database?.ok === true
    ) return 'available'
    return 'unknown'
  } catch {
    return 'unavailable'
  } finally {
    clearTimeout(timeout)
  }
}

export function usePublicServiceAvailability() {
  return useQuery({
    queryKey: ['public', 'backend-health'],
    queryFn: () => probePublicService(publicHealthUrl(API_BASE, window.location.href)),
    retry: false,
    staleTime: 60_000,
    refetchInterval: 120_000,
    refetchOnWindowFocus: true,
  })
}
