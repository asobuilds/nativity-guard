import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

interface MembershipAction {
  unitId: string
  membershipId: string
}

interface RevokeInput extends MembershipAction {
  reason: string
}

interface RejectInput extends MembershipAction {
  reason?: string
}

/**
 * All membership state-change mutations for a unit in one hook, so the
 * query-key invalidation lives in one place. Every successful action
 * refreshes the roster and the inbox.
 */
export function useMembershipActions(unitId: string | undefined) {
  const qc = useQueryClient()

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['unit-roster', unitId] })
    void qc.invalidateQueries({ queryKey: ['unit-inbox', unitId] })
    void qc.invalidateQueries({ queryKey: ['unit-governance', unitId] })
    void qc.invalidateQueries({ queryKey: ['unit-public-summary', unitId] })
  }

  const approve = useMutation({
    mutationFn: (input: MembershipAction) =>
      api.post(`/units/${input.unitId}/members/${input.membershipId}/approve`, {}),
    onSuccess: invalidate,
  })

  const reject = useMutation({
    mutationFn: (input: RejectInput) =>
      api.post(`/units/${input.unitId}/members/${input.membershipId}/reject`, {
        reason: input.reason ?? '',
      }),
    onSuccess: invalidate,
  })

  const revoke = useMutation({
    mutationFn: (input: RevokeInput) =>
      api.post(`/units/${input.unitId}/members/${input.membershipId}/revoke`, {
        reason: input.reason,
      }),
    onSuccess: invalidate,
  })

  const promote = useMutation({
    mutationFn: (input: MembershipAction) =>
      api.post(`/units/${input.unitId}/members/${input.membershipId}/promote`, {}),
    onSuccess: invalidate,
  })

  return { approve, reject, revoke, promote }
}
