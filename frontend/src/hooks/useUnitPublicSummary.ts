import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface UnitPublicSummary {
  unitId: string
  name: string
  type: string
  verified: boolean
  verifiedAt?: string
  verifiedDays: number
  formationDate?: string
  operationalRadius: number
  openCases: number
  totalCases: number
  recentCases: number
  activeMembers: number
  officers: number
}

export function useUnitPublicSummary(unitId: string | undefined) {
  return useQuery({
    queryKey: ['unit-public-summary', unitId],
    queryFn: () => api.get<UnitPublicSummary>(`/public/units/${unitId}/summary`),
    enabled: Boolean(unitId),
    staleTime: 60_000,
  })
}
