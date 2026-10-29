import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export type AgencyType = 'police' | 'nscdc' | 'fire' | 'frsc'

export interface AgencyAdmin {
  userId: string
  email: string
  firstName: string
  lastName: string
  phone: string
  rank: string
}

export interface AgencyTenant {
  id: string
  agencyType: AgencyType
  state: string
  lga: string
  divisionName: string
  createdAt: string
  admin?: AgencyAdmin
}

export function useAgencyTenants() {
  return useQuery({
    queryKey: ['agency-tenants'],
    queryFn: () => api.get<{ tenants: AgencyTenant[] }>('/super/agency-tenants'),
  })
}

export function useCreateAgencyTenant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { agencyType: AgencyType; state: string; lga: string; divisionName: string }) =>
      api.post('/super/agency-tenants', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agency-tenants'] }),
  })
}

export function useAssignAgencyAdmin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { tenantId: string; email: string; rank?: string }) =>
      api.post(`/super/agency-tenants/${input.tenantId}/admin`, {
        email: input.email,
        rank: input.rank || 'Division Admin',
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agency-tenants'] }),
  })
}
