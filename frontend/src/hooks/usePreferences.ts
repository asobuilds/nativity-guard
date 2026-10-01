import { useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/auth/AuthContext'
import { api } from '@/lib/apiClient'
import {
  applyPreferences,
  normalizePreferences,
  readPreferences,
  savePreferences,
  type Preferences,
} from '@/lib/preferences'

/**
 * Preferences come from two places:
 *   1. localStorage — applied instantly on boot, before the network
 *   2. the server   — the source of truth once signed in
 *
 * Every component that calls `usePreferences()` shares the same
 * React Query cache, so mounting it in App syncs everything.
 */
export function usePreferences() {
  const queryClient = useQueryClient()
  const { status } = useAuth()

  const query = useQuery<Preferences>({
    queryKey: ['preferences'],
    queryFn: async () => {
      if (status !== 'authenticated') return readPreferences()
      try {
        const res = await api.get<{ preferences: Partial<Preferences> }>(
          '/settings/preferences',
        )
        return normalizePreferences(res.preferences)
      } catch {
        return readPreferences()
      }
    },
    initialData: readPreferences(),
    staleTime: 5 * 60_000,
  })

  useEffect(() => {
    applyPreferences(query.data)
  }, [query.data])

  const updateMutation = useMutation({
    mutationFn: async (next: Partial<Preferences>) => {
      const merged = normalizePreferences({ ...query.data, ...next })
      savePreferences(merged)
      if (status === 'authenticated') {
        try {
          await api.put('/settings/preferences', {
            theme: merged.theme,
            textSize: merged.textSize,
            notifyAlerts: merged.notifyAlerts,
            notifyCaseUpdates: merged.notifyCaseUpdates,
            notifyCommunityReplies: merged.notifyCommunityReplies,
          })
        } catch {
          /* fall back to local only */
        }
      }
      return merged
    },
    onSuccess: (merged) => {
      queryClient.setQueryData(['preferences'], merged)
    },
  })

  return {
    preferences: query.data,
    update: (next: Partial<Preferences>) => updateMutation.mutate(next),
    saving: updateMutation.isPending,
  }
}