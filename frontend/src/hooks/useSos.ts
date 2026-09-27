import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
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
    mutationFn: (input: SendSosInput) => api.post<{ message: string; sos: SosAlert; escalationTime: string }>('/sos/send', input),
    onSuccess: () => client.invalidateQueries({ queryKey: sosKey }),
  })
}
