import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface RosterMember {
  membershipId: string
  userId: string
  firstName: string
  lastName: string
  email: string
  avatarPath?: string
  role: string
  status: 'active' | 'pending' | 'rejected' | 'revoked' | string
  isHeadAdmin: boolean
  joinedViaInvite: boolean
  acceptedAt?: string
  electedAt?: string
  termStartAt?: string
  termEndAt?: string
  consecutiveTerms: number
  verifiedAt?: string
  createdAt: string
}

export interface UnitRoster {
  members: RosterMember[]
  counts: {
    total: number
    admins: number
    officers: number
    pending: number
  }
}

export function useUnitRoster(unitId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ['unit-roster', unitId],
    queryFn: () => api.get<UnitRoster>(`/units/${unitId}/roster`),
    enabled: Boolean(unitId) && enabled,
    staleTime: 60_000,
  })
}
