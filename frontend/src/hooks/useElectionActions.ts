import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

interface OpenAdminElectionInput {
  unitId: string
  rotationGroup: 'A' | 'B'
}

interface OpenHeadAdminElectionInput {
  unitId: string
}

interface CastVoteInput {
  electionId: string
  candidateId: string
}

interface CloseElectionInput {
  electionId: string
}

interface FillVacancyInput {
  seatId: string
}

/**
 * All election mutations in one hook so invalidations happen in one place.
 * Every successful action invalidates the whole governance cache for the
 * unit — cheap, correct, and never stale.
 */
export function useElectionActions(unitId: string | undefined) {
  const qc = useQueryClient()

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['unit-governance', unitId] })
    void qc.invalidateQueries({ queryKey: ['unit-roster', unitId] })
  }

  const openAdminElection = useMutation({
    mutationFn: (input: OpenAdminElectionInput) =>
      api.post(`/units/${input.unitId}/elections`, {
        rotationGroup: input.rotationGroup,
      }),
    onSuccess: invalidate,
  })

  const openHeadAdminElection = useMutation({
    mutationFn: (input: OpenHeadAdminElectionInput) =>
      api.post(`/units/${input.unitId}/head-admin-elections`, {}),
    onSuccess: invalidate,
  })

  const castVote = useMutation({
    mutationFn: (input: CastVoteInput) =>
      api.post(`/elections/${input.electionId}/vote`, {
        candidateId: input.candidateId,
      }),
    onSuccess: invalidate,
  })

  const closeElection = useMutation({
    mutationFn: (input: CloseElectionInput) =>
      api.post(`/elections/${input.electionId}/close`, {}),
    onSuccess: invalidate,
  })

  const fillVacancy = useMutation({
    mutationFn: (input: FillVacancyInput) =>
      api.post(`/elections/seats/${input.seatId}/fill`, {}),
    onSuccess: invalidate,
  })

  return {
    openAdminElection,
    openHeadAdminElection,
    castVote,
    closeElection,
    fillVacancy,
  }
}
