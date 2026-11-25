import { describe, expect, it, vi } from 'vitest'
import { probePublicService, publicHealthUrl } from './usePublicServiceAvailability'

function healthResponse(status: 'ok' | 'degraded' | 'down', dbOK: boolean, httpStatus = 200) {
  return new Response(JSON.stringify({ status, checks: { database: { ok: dbOK } } }), {
    status: httpStatus,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('public backend health resolution', () => {
  it('uses the actual API origin, not the website host', () => {
    expect(publicHealthUrl(
      'https://nativityguard-backend.onrender.com/api/v1',
      'https://nativityguard.com/',
    )).toBe('https://nativityguard-backend.onrender.com/health')
  })

  it('supports a same-origin reverse proxy configuration', () => {
    expect(publicHealthUrl('/api/v1', 'https://nativityguard.com/'))
      .toBe('https://nativityguard.com/health')
  })
})

describe('public availability probe', () => {
  it('recognises healthy database and scheduler-degraded states', async () => {
    const fetcher = vi.fn().mockResolvedValue(healthResponse('ok', true)) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', fetcher)).toBe('available')
    const degraded = vi.fn().mockResolvedValue(healthResponse('degraded', true)) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', degraded)).toBe('available')
  })

  it('reports an unreachable service for 503, down status and network errors', async () => {
    const serverError = vi.fn().mockResolvedValue(healthResponse('down', false, 503)) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', serverError)).toBe('unavailable')
    const rejected = vi.fn().mockRejectedValue(new Error('Cannot connect')) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', rejected)).toBe('unavailable')
  })

  it('does not treat a 404/static website or unexpected HTML as evidence of downtime', async () => {
    const missing = vi.fn().mockResolvedValue(new Response('Not found', { status: 404 })) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', missing)).toBe('unknown')
    const html = vi.fn().mockResolvedValue(new Response('<html>index</html>', { status: 200 })) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', html)).toBe('unknown')
  })

  it('aborts a stalled health probe rather than leaving the visitor waiting indefinitely', async () => {
    const stalled = vi.fn().mockImplementation((_url: string, init: RequestInit) =>
      new Promise((_resolve, reject) => {
        init.signal?.addEventListener('abort', () => reject(new Error('aborted')), { once: true })
      }),
    ) as unknown as typeof fetch
    expect(await probePublicService('https://example.test/health', stalled, 5)).toBe('unavailable')
  })
})
