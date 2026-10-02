import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, tokenStore, UNAUTHORIZED_EVENT } from '@/lib/apiClient'
import { queryClient } from '@/lib/queryClient'
import { normaliseRole } from '@/lib/role'
import type { LoginResponse, ProfileResponse, Role, User } from '@/types/api'

type AuthStatus = 'loading' | 'authenticated' | 'anonymous'

interface AuthContextValue {
  user: User | null
  /** The canonical role, or `null` when the account carries one we cannot name. */
  role: Role | null
  /**
   * The role exactly as the API returned it. Kept alongside `role` so an
   * unrecognised value can be *shown* to the user and fixed by an administrator,
   * rather than silently dropped into looking like "no role at all".
   */
  rawRole: string | null
  status: AuthStatus
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  refreshProfile: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<AuthStatus>('loading')

  const logout = useCallback(() => {
    queryClient.clear()
    tokenStore.clear()
    setUser(null)
    setStatus('anonymous')
  }, [])

  // Restore a session on boot if a token exists.
  useEffect(() => {
    let cancelled = false

    async function restore() {
      if (!tokenStore.get()) {
        setStatus('anonymous')
        return
      }
      try {
        const { user: fresh } = await api.get<ProfileResponse>('/auth/profile')
        if (!cancelled) {
          setUser(fresh)
          setStatus('authenticated')
        }
      } catch {
        if (!cancelled) logout()
      }
    }

    void restore()
    return () => {
      cancelled = true
    }
  }, [logout])

  // Any 401 from the API client tears the session down.
  useEffect(() => {
    const onUnauthorized = () => logout()
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized)
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized)
  }, [logout])

  const login = useCallback(async (email: string, password: string) => {
    // Anonymous: sign-in must not send a stale token, and a rejected login must
    // not clear an existing session (see api.postAnonymous).
    const result = await api.postAnonymous<LoginResponse>('/auth/login', { email, password })
    // Cached records belong to the previous authenticated identity.
    queryClient.clear()
    tokenStore.set(result.token)
    // The refresh token is what keeps this session alive past the access token's
    // 24h. Not storing it is why every session used to die at its first expiry,
    // with no warning and whatever the user was doing thrown away.
    tokenStore.setRefresh(result.refreshToken)
    // Login returns a partial user; fetch the full profile for status/timestamps.
    try {
      const { user: fresh } = await api.get<ProfileResponse>('/auth/profile')
      setUser(fresh)
    } catch {
      setUser(result.user as User)
    }
    setStatus('authenticated')
  }, [])
  const refreshProfile = useCallback(async () => {
    const { user: fresh } = await api.get<ProfileResponse>('/auth/profile')
    setUser(fresh)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      // Normalise at the boundary, so every consumer downstream may assume a
      // canonical role or `null` — and `null` is a state the UI renders rather than
      // a value any guard is allowed to guess a destination from.
      role: normaliseRole(user?.role),
      rawRole: user?.role ?? null,
      status,
      login,
      logout,
      refreshProfile,
    }),
    [user, status, login, logout, refreshProfile],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within <AuthProvider>')
  return ctx
}
