import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, postQueued } from '@/lib/apiClient'
import type { SendSosInput, SosAlert } from '@/types/api'

const sosKey = ['sos', 'mine'] as const

export function useMySos(enabled = true) {
  return useQuery({
    queryKey: sosKey,
    queryFn: () => api.get<{ alerts: SosAlert[] }>('/sos/my'),
    enabled,
    select: (data) => data.alerts,
    refetchInterval: 15_000,
  })
}

export function useSendSos() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: async (input: SendSosInput) => {
      // Offline-safe: if the network is down, save the SOS locally and
      // report a queued state to the caller. The banner will surface it.
      const result = await postQueued<{ message: string; sos: SosAlert; escalationTime: string }>('/sos/send', input)
      if (result && typeof result === 'object' && 'queued' in result) {
        return { message: 'SOS queued — will send when back online.', sos: null as unknown as SosAlert, escalationTime: '', queued: true as const }
      }
      return result
    },
    onSuccess: () => {
      // Invalidate BOTH the narrow and broad keys so any surface
      // (SosPage history, dashboard, notification bell) refetches.
      void client.invalidateQueries({ queryKey: sosKey })
      void client.invalidateQueries({ queryKey: ['sos'] })
    },
  })
}
