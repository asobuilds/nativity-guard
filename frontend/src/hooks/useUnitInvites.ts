import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface UnitInvite {
  id: string
  scope: string
  status: 'pending' | 'revoked' | 'expired' | 'exhausted' | string
  maxUses: number
  useCount: number
  expiresAt?: string
  revokedAt?: string
  revokedReason?: string
  createdBy: string
  createdAt: string
}

interface InvitesResponse {
  invites: UnitInvite[]
}

interface CreateInviteResponse {
  invite: {
    id: string
    scope: string
    unitId: string
    code: string
    expiresAt?: string
    maxUses: number
  }
  warning: string
}

export function useUnitInvites(unitId: string | undefined) {
  return useQuery({
    queryKey: ['unit-invites', unitId],
    queryFn: () => api.get<InvitesResponse>(`/units/${unitId}/invites`),
    select: (d) => d.invites,
    enabled: Boolean(unitId),
    staleTime: 30_000,
  })
}

export function useInviteActions(unitId: string | undefined) {
  const qc = useQueryClient()

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['unit-invites', unitId] })
  }

  const create = useMutation({
    mutationFn: (body: { expiresInHours?: number; maxUses?: number }) =>
      api.post<CreateInviteResponse>(`/units/${unitId}/invites`, body),
    onSuccess: invalidate,
  })

  const revoke = useMutation({
    mutationFn: (inviteId: string) =>
      api.post(`/units/${unitId}/invites/${inviteId}/revoke`, {}),
    onSuccess: invalidate,
  })

  return { create, revoke }
}
