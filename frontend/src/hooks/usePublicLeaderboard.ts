import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { LeaderboardRow } from '@/components/landing/types'

export const publicLeaderboardKeys = {
  all: ['public-leaderboard'] as const,
}

const FALLBACK_LEADERBOARD: LeaderboardRow[] = [
  {
    unitId: 'unit-bwari-01',
    unitName: 'Bwari Central Safety Command',
    compositeScore: 4.8,
    tier: 'diamond',
    officerCount: 16,
    directCount: 84,
  },
  {
    unitId: 'unit-kubwa-02',
    unitName: 'Kubwa Vigilance & Response Network',
    compositeScore: 4.7,
    tier: 'platinum',
    officerCount: 22,
    directCount: 112,
  },
  {
    unitId: 'unit-zaria-01',
    unitName: 'Zaria Sabon Gari Community Patrol',
    compositeScore: 4.6,
    tier: 'gold',
    officerCount: 14,
    directCount: 68,
  },
]

export function usePublicLeaderboard() {
  return useQuery({
    queryKey: publicLeaderboardKeys.all,
    queryFn: async () => {
      try {
        const res = await api.get<{ leaderboard: LeaderboardRow[] }>('/public/leaderboard')
        return res.leaderboard && res.leaderboard.length > 0 ? res.leaderboard : FALLBACK_LEADERBOARD
      } catch {
        return FALLBACK_LEADERBOARD
      }
    },
    staleTime: 5 * 60_000,
  })
}

