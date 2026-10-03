import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

interface OfficerCompliance {
  membershipId: string
  userId: string
  firstName: string
  lastName: string
  email: string
  avatarPath?: string
  weeksFiled: number
  weeksTotal: number
  complianceRate: number
  weekCounts: number[]
}

export interface UnitCompliance {
  weeks: string[]
  officers: OfficerCompliance[]
}

export function useUnitCompliance(unitId: string | undefined) {
  return useQuery({
    queryKey: ['unit-compliance', unitId],
    queryFn: () => api.get<UnitCompliance>(`/units/${unitId}/compliance`),
    enabled: Boolean(unitId),
    staleTime: 60_000,
  })
}
