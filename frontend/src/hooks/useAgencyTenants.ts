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
  unitId?: string
  createdAt: string
  admin?: AgencyAdmin
}

export interface AgencyObserver {
  id: string
  userId: string
  tenantId: string
  rank: string
  createdAt: string
  email: string
  phone: string
  firstName: string
  lastName: string
  status: string
}

export interface CreateAgencyObserverInput {
  email: string
  phone: string
  firstName: string
  lastName: string
  password: string
  rank?: string
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

/** Agency observers are always returned from the caller's server-resolved tenant. */
export function useAgencyObservers() {
  return useQuery({
    queryKey: ['agency-observers'],
    queryFn: () => api.get<{ observers: AgencyObserver[] }>('/agency/observers'),
    staleTime: 30_000,
  })
}

/** Creates an observer inside the authenticated division admin's tenant only. */
export function useCreateAgencyObserver() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAgencyObserverInput) =>
      api.post('/agency/observers', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agency-observers'] }),
  })
}

/** Link a division to one active, verified, jurisdiction-matched operational unit. */
export function useLinkAgencyTenantUnit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { tenantId: string; unitId: string }) =>
      api.post(`/super/agency-tenants/${input.tenantId}/unit`, { unitId: input.unitId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agency-tenants'] }),
  })
}
