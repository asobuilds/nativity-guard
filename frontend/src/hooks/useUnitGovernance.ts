import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface GovernanceAdmin {
  membershipId: string
  userId: string
  firstName: string
  lastName: string
  email: string
  avatarPath?: string
  isHeadAdmin: boolean
  electedAt?: string
  termStartAt?: string
  termEndAt?: string
  consecutiveTerms: number
  coolingOffUntil?: string
}

export interface AdminSeat {
  id: string
  unitId: string
  electionId: string
  seatNumber: number
  rotationGroup: string
  memberId?: string
  termStart: string
  termEnd: string
  status: 'active' | 'vacant' | string
  electedAt: string
  vacatedAt?: string
}

export interface Election {
  id: string
  unitId: string
  electionType: 'admin' | 'head_admin' | string
  cycleNumber: number
  rotationGroup: string
  seatCount: number
  eligibleVoterCount: number
  quorumCount: number
  extendedOnce: boolean
  quorumMet: boolean
  termStart: string
  termEnd: string
  votingStartsAt?: string
  votingEndsAt?: string
  resultFinalizedAt?: string
  status: string
  createdAt: string
}

export interface RevocationCycle {
  id: string
  targetMembershipId: string
  targetRole: string
  cycleType: string
  status: string
  eligibleMemberCount: number
  quorumRequired: number
  requiredVotes: number
  forVotes: number
  againstVotes: number
  abstainVotes: number
  totalVotes: number
  headAdminApproved: boolean
  reason: string
  openedAt: string
  closedAt?: string
}

export interface UnitGovernance {
  unitId: string
  admins: GovernanceAdmin[]
  adminCount: number
  headAdminCount: number
  seats: AdminSeat[]
  openElections: Election[]
  recentElections: Election[]
  openRevocations: RevocationCycle[]
}

export function useUnitGovernance(unitId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ['unit-governance', unitId],
    queryFn: () => api.get<UnitGovernance>(`/units/${unitId}/governance`),
    enabled: Boolean(unitId) && enabled,
    staleTime: 30_000,
  })
}
