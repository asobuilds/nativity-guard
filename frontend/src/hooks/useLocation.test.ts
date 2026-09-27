import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  initialLocationState,
  ipFallback,
  locationReducer,
  readLocationCache,
  writeLocationCache,
  type LocationState,
} from './useLocation'

/**
 * The location hook is built on a pure state machine (`locationReducer`) plus
 * cache helpers, so it can be pinned here in this repo's node-only test
 * environment — there is no DOM runner installed. These cases mirror the three
 * behaviours the hook must surface to callers.
 */

beforeEach(() => {
  // Mock navigator.geolocation + the Permissions API so the environment is
  // realistic even though the reducer itself never reaches for them.
  vi.stubGlobal('navigator', {
    ...navigator,
    geolocation: {
      getCurrentPosition: vi.fn(),
      watchPosition: vi.fn(),
      clearWatch: vi.fn(),
    },
    permissions: {
      query: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    },
  })

  // A session-scoped cache so `initialLocationState` is deterministic per test.
  const store: Record<string, string> = {}
  vi.stubGlobal('sessionStorage', {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = value
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      for (const key of Object.keys(store)) delete store[key]
    },
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const COORDS = { latitude: 6.5244, longitude: 3.3792, accuracy: 18 }

describe('useLocation state machine', () => {
  it('returns null on mount when permission is "prompt"', () => {
    const start = initialLocationState()
    expect(start.latitude).toBeNull()
    expect(start.longitude).toBeNull()

    const resolved = locationReducer(start, {
      type: 'permission_resolved',
      permission: 'prompt',
    })
    expect(resolved.latitude).toBeNull()
    expect(resolved.permission).toBe('prompt')
    expect(resolved.loading).toBe(false)
  })

  it('request() updates state on success', () => {
    const start = initialLocationState()

    const fetching = locationReducer(start, { type: 'fetch_start' })
    expect(fetching.loading).toBe(true)
    expect(fetching.error).toBeNull()

    const done = locationReducer(fetching, { type: 'position_success', coords: COORDS })
    expect(done.latitude).toBe(COORDS.latitude)
    expect(done.longitude).toBe(COORDS.longitude)
    expect(done.accuracy).toBe(COORDS.accuracy)
    expect(done.loading).toBe(false)
    expect(done.permission).toBe('granted')
    expect(done.error).toBeNull()
  })

  it('denial path sets permission="denied"', () => {
    const start = initialLocationState()

    const fetching = locationReducer(start, { type: 'fetch_start' })
    const denied = locationReducer(fetching, {
      type: 'position_error',
      message: 'User denied Geolocation',
    })
    expect(denied.permission).toBe('denied')
    expect(denied.latitude).toBeNull()
    expect(denied.longitude).toBeNull()
    expect(denied.loading).toBe(false)
    expect(denied.error).toBe('User denied Geolocation')
  })

  it('keeps a denied state when permissions resolve to denied on mount', () => {
    const start: LocationState = initialLocationState()
    const denied = locationReducer(start, {
      type: 'permission_resolved',
      permission: 'denied',
      error: 'Location permission is denied.',
    })
    expect(denied.permission).toBe('denied')
    expect(denied.latitude).toBeNull()
    expect(denied.error).toBe('Location permission is denied.')
  })
})

describe('useLocation cache', () => {
  it('caches a successful fix so a route change does not re-prompt', () => {
    writeLocationCache({
      latitude: COORDS.latitude,
      longitude: COORDS.longitude,
      accuracy: COORDS.accuracy,
      permission: 'granted',
    })

    expect(readLocationCache()?.latitude).toBe(COORDS.latitude)
    expect(readLocationCache()?.permission).toBe('granted')

    // A fresh mount reads the cache back as the initial state.
    const restored = initialLocationState()
    expect(restored.latitude).toBe(COORDS.latitude)
    expect(restored.longitude).toBe(COORDS.longitude)
    expect(restored.permission).toBe('granted')
  })

  it('starts empty when the cache is absent', () => {
    const start = initialLocationState()
    expect(start.latitude).toBeNull()
    expect(start.longitude).toBeNull()
    expect(start.permission).toBe('unavailable')
  })
})

describe('useLocation ipFallback', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
    // AbortController is native in Node 18+; stub if unavailable.
    if (typeof AbortController === 'undefined') {
      vi.stubGlobal('AbortController', class {
        readonly signal = {}
        abort() {}
      })
    }
  })

  it('returns cached result when the cache is still fresh', async () => {
    sessionStorage.setItem(
      'cs.ipLocation',
      JSON.stringify({ latitude: 6.5244, longitude: 3.3792, ts: Date.now() }),
    )

    const result = await ipFallback()
    expect(result).toEqual({ latitude: 6.5244, longitude: 3.3792 })
    expect(fetch).not.toHaveBeenCalled()
  })

  it('fetches and caches when no valid cache exists', async () => {
    const mockFetch = vi.mocked(fetch)
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ latitude: 6.5244, longitude: 3.3792 }),
    } as Response)

    const result = await ipFallback()
    expect(result).toEqual({ latitude: 6.5244, longitude: 3.3792 })
    expect(mockFetch).toHaveBeenCalledWith('https://ipapi.co/json/', expect.objectContaining({ signal: expect.anything() }))

    const stored = JSON.parse(sessionStorage.getItem('cs.ipLocation')!)
    expect(stored.latitude).toBe(6.5244)
    expect(stored.longitude).toBe(3.3792)
  })

  it('returns null when the response is not ok', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({ ok: false } as Response)

    const result = await ipFallback()
    expect(result).toBeNull()
  })

  it('returns null on fetch failure', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new Error('network'))

    const result = await ipFallback()
    expect(result).toBeNull()
  })

  it('re-fetches when the cached result is stale (older than 30 min)', async () => {
    const staleTs = Date.now() - 31 * 60 * 1000
    sessionStorage.setItem(
      'cs.ipLocation',
      JSON.stringify({ latitude: 1, longitude: 2, ts: staleTs }),
    )

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ latitude: 6.5244, longitude: 3.3792 }),
    } as Response)

    const result = await ipFallback()
    expect(result).toEqual({ latitude: 6.5244, longitude: 3.3792 })
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('returns null when the response lacks latitude/longitude', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ city: 'Lagos' }),
    } as Response)

    const result = await ipFallback()
    expect(result).toBeNull()
  })
})
