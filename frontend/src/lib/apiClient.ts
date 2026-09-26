/**
 * Typed fetch client for the Nativity Guard API.
 *
 * - Injects `Authorization: Bearer <token>` when a session exists.
 * - Normalises errors into `ApiError`.
 * - On 401 it clears the session and emits `cs:unauthorized` so AuthContext can
 *   route the user back to login (see src/auth/AuthContext.tsx).
 */

import type { RefreshResponse } from '@/types/api'

const TOKEN_KEY = 'cs.token'
const REFRESH_KEY = 'cs.refresh'

function resolveBase(): string {
  const raw = (import.meta.env.VITE_API_URL ?? '').trim()
  if (!raw) return '/api/v1'
  const origin = raw.replace(/\/+$/, '')
  return origin.endsWith('/api/v1') ? origin : `${origin}/api/v1`
}

export const API_BASE = resolveBase()

export const UNAUTHORIZED_EVENT = 'cs:unauthorized'

export class ApiError extends Error {
  readonly status: number
  readonly body: unknown

  constructor(status: number, message: string, body?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }

  /** True when the failure is a connectivity problem rather than an HTTP error. */
  static isNetwork(error: unknown): boolean {
    return error instanceof ApiError && error.status === 0
  }
}

function readKey(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeKey(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* storage unavailable (private mode) — session stays in memory only */
  }
}

function removeKey(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    /* ignore */
  }
}

/**
 * Both halves of a session.
 *
 * The access token is short-lived; the refresh token is what keeps a signed-in
 * user signed in past it. The two are stored and cleared together — clearing one
 * without the other is how you get a session that can neither refresh nor end.
 */
export const tokenStore = {
  get: () => readKey(TOKEN_KEY),
  set: (token: string) => writeKey(TOKEN_KEY, token),
  getRefresh: () => readKey(REFRESH_KEY),
  setRefresh: (token: string) => writeKey(REFRESH_KEY, token),
  clear(): void {
    removeKey(TOKEN_KEY)
    removeKey(REFRESH_KEY)
  },
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
  /** Skip auth header + 401 handling (used by login). */
  anonymous?: boolean
}

/**
 * Turn a failed response into a message **and** keep the parsed body.
 *
 * The body matters: several endpoints refuse with a 409 that carries the reason in
 * structured form — `{ error, status }` from `submit-review`, `{ error, update }`
 * from a duplicate weekly update. Discarding it leaves the UI able to say only
 * "conflict", when the contract is handing over exactly what it needs to correct
 * itself.
 */
async function readError(response: Response): Promise<{ message: string; body: unknown }> {
  try {
    const data = (await response.json()) as { error?: string; message?: string }
    return { message: data?.error ?? data?.message ?? response.statusText, body: data }
  } catch {
    return {
      message: response.statusText || `Request failed (${response.status})`,
      body: undefined,
    }
  }
}

/**
 * The in-flight refresh, shared by every caller.
 *
 * Rotation makes concurrency a correctness problem rather than an efficiency one.
 * If three requests each 401'd and each spent the refresh token, the second would
 * present one the first had already consumed and be rejected as revoked — signing
 * the user out at the exact moment the app was trying to keep them in.
 */
let refreshInFlight: Promise<string | null> | null = null

/**
 * Exchange the refresh token for a new pair, or `null` if it will not.
 *
 * Called with bare `fetch`, deliberately: routing it through `request` would let a
 * 401 on the refresh recurse into another refresh.
 */
async function performRefresh(): Promise<string | null> {
  const refreshToken = tokenStore.getRefresh()
  if (!refreshToken) return null

  try {
    const response = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    })
    if (!response.ok) return null

    const payload = (await response.json()) as Partial<RefreshResponse>
    if (!payload.token) return null

    tokenStore.set(payload.token)
    // Rotation: the token we just sent is spent. Failing to store its replacement
    // means the *next* refresh presents a dead token and signs the user out.
    if (payload.refreshToken) tokenStore.setRefresh(payload.refreshToken)
    return payload.token
  } catch {
    // Network failure or an unparseable body — either way, no new token.
    return null
  }
}

/**
 * Share the in-flight attempt, and clear it once it settles so a later expiry can
 * start a fresh one — a cached-forever promise would silently stop refreshing.
 */
function refreshAccessToken(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight
  refreshInFlight = performRefresh().finally(() => {
    refreshInFlight = null
  })
  return refreshInFlight
}

/** Build and send one attempt, reading the token fresh so a retry picks up a new one. */
async function send(
  path: string,
  method: string,
  body: unknown,
  signal: AbortSignal | undefined,
  anonymous: boolean,
): Promise<Response> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const token = tokenStore.get()
  if (token && !anonymous) headers.Authorization = `Bearer ${token}`

  try {
    return await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      signal,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause
    throw new ApiError(0, 'Network unavailable — check your connection.', cause)
  }
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal, anonymous = false } = options

  let response = await send(path, method, body, signal, anonymous)

  // A 401 on an authenticated call usually means the access token aged out, not
  // that the session is over. Try the refresh token once and replay — retrying
  // more than once would loop, since a second 401 means the new token was
  // rejected too, which is a genuine sign-out.
  if (response.status === 401 && !anonymous) {
    const refreshed = await refreshAccessToken()
    if (refreshed) response = await send(path, method, body, signal, anonymous)
  }

  if (response.status === 401 && !anonymous) {
    tokenStore.clear()
    // Guarded so the client also works outside a DOM (tests, future SSR).
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT))
    }
    const { message } = await readError(response)
    throw new ApiError(401, message)
  }

  if (!response.ok) {
    const { message, body: errorBody } = await readError(response)
    throw new ApiError(response.status, message, errorBody)
  }

  if (response.status === 204) return undefined as T

  const text = await response.text()
  if (!text) return undefined as T
  return JSON.parse(text) as T
}

/**
 * `postAnonymous` exists for sign-in: a login request must not carry a stale
 * bearer token, and a 401 from *bad credentials* must not tear down a session
 * the user already has — only an authenticated request can do that.
 */
/**
 * Multipart upload helper — uses the same 401/refresh pipeline as `request`.
 *
 * Do NOT set Content-Type; the browser sets it with the correct multipart boundary.
 */
async function requestUpload<T>(
  path: string,
  file: File,
  fieldName: string,
  anonymous: boolean,
): Promise<T> {
  const formData = new FormData()
  formData.append(fieldName, file)

  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = tokenStore.get()
  if (token && !anonymous) headers.Authorization = `Bearer ${token}`

  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      method: 'POST',
      headers,
      body: formData,
    })
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause
    throw new ApiError(0, 'Network unavailable — check your connection.', cause)
  }

  // Same 401 handling as request(): refresh once, replay, then clear session.
  if (response.status === 401 && !anonymous) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      const retryHeaders: Record<string, string> = { Accept: 'application/json' }
      const newToken = tokenStore.get()
      if (newToken) retryHeaders.Authorization = `Bearer ${newToken}`
      response = await fetch(`${API_BASE}${path}`, {
        method: 'POST',
        headers: retryHeaders,
        body: formData,
      })
    }
  }

  if (response.status === 401 && !anonymous) {
    tokenStore.clear()
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT))
    }
    const { message } = await readError(response)
    throw new ApiError(401, message)
  }

  if (!response.ok) {
    const { message, body: errorBody } = await readError(response)
    throw new ApiError(response.status, message, errorBody)
  }

  if (response.status === 204) return undefined as T

  const text = await response.text()
  if (!text) return undefined as T
  return JSON.parse(text) as T
}

export const api = {
	get: <T>(path: string, signal?: AbortSignal) => request<T>(path, { method: 'GET', signal }),
	post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
	postAnonymous: <T>(path: string, body?: unknown) =>
		request<T>(path, { method: 'POST', body, anonymous: true }),
	put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body }),
	patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body }),
	delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
	upload: <T>(path: string, file: File, fieldName = 'file'): Promise<T> =>
		requestUpload<T>(path, file, fieldName, false),
	uploadAnonymous: <T>(path: string, file: File, fieldName = 'file'): Promise<T> =>
		requestUpload<T>(path, file, fieldName, true),
}

/**
 * Constructs a full media URL from a relative path returned by the backend.
 *
 * The backend returns avatarPath/coverPath as relative paths like
 * "avatars/2026/09/hash.jpg". The frontend must prefix these with the
 * backend's public media base (e.g. Supabase storage public URL) to render
 * them correctly.
 *
 * If the path is already a full URL (starts with http:// or https://), it is
 * returned as-is. If VITE_MEDIA_BASE is not set, the original path is returned
 * as a graceful fallback (may not render if the frontend and media are on
 * different origins).
 */
export function mediaURL(path: string | null | undefined): string | null {
	if (!path) return null
	// If it's already a full URL, use it as-is
	if (/^https?:\/\//i.test(path)) return path
	// Otherwise prefix with the backend's public media base
	const base = (import.meta.env.VITE_MEDIA_BASE ?? '').replace(/\/+$/, '')
	if (!base) return path // graceful fallback
	return `${base}/${path.replace(/^\/+/, '')}`
}
