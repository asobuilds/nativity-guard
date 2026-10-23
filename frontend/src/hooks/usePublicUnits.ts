import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'

export const publicUnitKeys = {
  all: ['public-units'] as const,
}

/** Only publish records returned by the live API. Never substitute sample patrol units. */
export function usePublicUnits() {
  return useQuery({
    queryKey: publicUnitKeys.all,
    queryFn: async () => {
      const res = await api.get<{ units: SecurityUnit[] }>('/public/units')
      return res.units ?? []
    },
    staleTime: 5 * 60_000,
    networkMode: 'offlineFirst',
  })
}
