import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'

/**
 * The backend SecurityUnit model carries verification metadata that the
 * frontend `types/api.ts` does not yet declare. Extend it locally rather
 * than touching the shared type.
 */
export type ExtendedUnit = SecurityUnit & {
  verificationSubmittedAt?: string
  verifiedAt?: string
  verifiedBy?: string
  verificationNotes?: string
}

interface RecordMember {
  membershipId: string
  userId: string
  firstName: string
  lastName: string
  email: string
  avatarPath?: string
  role: string
  status: string
  isHeadAdmin: boolean
  joinedViaInvite: boolean
  acceptedAt?: string
  electedAt?: string
  termStartAt?: string
  termEndAt?: string
  verifiedAt?: string
  createdAt: string
}

interface RecordSeat {
  id: string
  seatNumber: number
  rotationGroup: string
  memberId?: string
  termStart: string
  termEnd: string
  status: string
  electedAt: string
  vacatedAt?: string
}

interface RecordElection {
  id: string
  electionType: string
  seatCount: number
  eligibleVoterCount: number
  quorumCount: number
  quorumMet: boolean
  extendedOnce: boolean
  termStart: string
  termEnd: string
  votingStartsAt?: string
  votingEndsAt?: string
  resultFinalizedAt?: string
  status: string
  createdAt: string
}

interface RecordRevocation {
  id: string
  targetMembershipId: string
  targetRole: string
  cycleType: string
  status: string
  forVotes: number
  againstVotes: number
  abstainVotes: number
  requiredVotes: number
  eligibleMemberCount: number
  headAdminApproved: boolean
  reason: string
  openedAt: string
  closedAt?: string
}

interface RecordAuditEntry {
  id: string
  userId: string
  action: string
  entityType: string
  entityId: string
  oldValue?: string
  newValue?: string
  ipAddress: string
  userAgent: string
  timestamp: string
  user?: {
    id: string
    email: string
    firstName: string
    lastName: string
  }
}

export interface UnitRecord {
  unit: ExtendedUnit
  counts: {
    active: number
    pending: number
    admins: number
    officers: number
    headAdmin: number
  }
  members: RecordMember[]
  seats: RecordSeat[]
  openElections: RecordElection[]
  recentElections: RecordElection[]
  revocations: RecordRevocation[]
  cases: { open: number; total: number }
  auditLog: RecordAuditEntry[]
}

export function useUnitRecord(unitId: string | undefined) {
  return useQuery({
    queryKey: ['admin-unit-record', unitId],
    queryFn: () => api.get<UnitRecord>(`/admin/units/${unitId}/record`),
    enabled: Boolean(unitId),
    staleTime: 30_000,
  })
}
