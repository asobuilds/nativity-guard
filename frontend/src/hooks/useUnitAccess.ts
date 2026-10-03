import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface UnitAccess {
  isMember: boolean
  role: string | null
  status: string | null
  isHeadAdmin: boolean
  membershipId: string | null
}

export function useUnitAccess(unitId: string | undefined) {
  return useQuery({
    queryKey: ['unit-access', unitId],
    queryFn: () => api.get<UnitAccess>(`/units/${unitId}/access`),
    enabled: Boolean(unitId),
    staleTime: 60_000,
  })
}
