import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, postQueued } from '@/lib/apiClient'
import type { SendSosInput, SosAlert } from '@/types/api'

const sosKey = ['sos', 'mine'] as const

export interface SosResponder {
  id: string
  sosId: string
  unitId: string
  role: 'primary' | 'support'
  acceptedBy: string
  acceptedAt: string
  unit?: { id: string; name: string; type?: string }
}

export interface SosDetail extends SosAlert {
  severity?: 'normal' | 'critical'
  dispatchState?: 'open' | 'locked' | 'multi_responder' | 'resolved'
  acceptedByUnitId?: string
  acceptedAt?: string
  jurisdictionState?: string
  jurisdictionLga?: string
  responders?: SosResponder[]
}

export function useMySos(enabled = true) {
  return useQuery({
    queryKey: sosKey,
    queryFn: () => api.get<{ alerts: SosAlert[] }>('/sos/my'),
    enabled,
    select: (data) => data.alerts,
    refetchInterval: 15_000,
  })
}

export function useSosDetail(id: string | undefined) {
  return useQuery({
    queryKey: ['sos', 'detail', id],
    queryFn: () => api.get<{ sos: SosDetail; responders?: SosResponder[] }>('/sos/' + id),
    select: (d) => ({ ...d.sos, responders: d.responders ?? d.sos.responders ?? [] }),
    enabled: Boolean(id),
    refetchInterval: 10_000,
  })
}

export function useSendSos() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: async (input: SendSosInput) => {
      const result = await postQueued<{ message: string; sos: SosAlert; escalationTime: string }>('/sos/send', input)
      if (result && typeof result === 'object' && 'queued' in result) {
        return { message: 'SOS queued - will send when back online.', sos: null as unknown as SosAlert, escalationTime: '', queued: true as const }
      }
      return result
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: sosKey })
      void client.invalidateQueries({ queryKey: ['sos'] })
    },
  })
}

export function useAcceptSos() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (sosId: string) =>
      api.post<{ message: string; responder: SosResponder; dispatchState: string; responderCount: number }>(
        '/sos/' + sosId + '/accept',
        {},
      ),
    onSuccess: (_data, sosId) => {
      void client.invalidateQueries({ queryKey: ['sos', 'detail', sosId] })
      void client.invalidateQueries({ queryKey: ['sos'] })
    },
  })
}

export function useReleaseSos() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (sosId: string) =>
      api.post<{ message: string; responderCount: number; dispatchState: string }>(
        '/sos/' + sosId + '/release',
        {},
      ),
    onSuccess: (_data, sosId) => {
      void client.invalidateQueries({ queryKey: ['sos', 'detail', sosId] })
      void client.invalidateQueries({ queryKey: ['sos'] })
    },
  })
}

export function useAssignSos() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: { sosId: string; unitId: string; role?: 'primary' | 'support' }) =>
      api.post<{ message: string; responder: SosResponder; dispatchState: string; responderCount: number }>(
        '/sos/' + input.sosId + '/assign',
        { unitId: input.unitId, role: input.role },
      ),
    onSuccess: (_data, variables) => {
      void client.invalidateQueries({ queryKey: ['sos', 'detail', variables.sosId] })
      void client.invalidateQueries({ queryKey: ['sos'] })
    },
  })
}