import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { PublicCase } from '@/components/landing/types'

export const publicCaseKeys = {
  all: ['public-cases'] as const,
}

/** Never substitute fictional incidents for live safety data. */
export function usePublicCases() {
  return useQuery({
    queryKey: publicCaseKeys.all,
    queryFn: async () => {
      const res = await api.get<{ cases: PublicCase[] }>('/public/cases')
      return res.cases ?? []
    },
    staleTime: 60_000,
    networkMode: 'offlineFirst',
  })
}
