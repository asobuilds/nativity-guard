import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface LGAMessage {
  id: string
  state: string
  lga: string
  category: 'intel' | 'alert' | 'coordination' | 'request' | 'general'
  priority: 'normal' | 'urgent'
  title: string
  body: string
  authorUserId: string
  authorUnitId: string
  authorName: string
  authorUnit: string
  expiresAt?: string
  createdAt: string
}

export interface LGAChannel {
  lga: string
  state: string
}

interface ChannelResponse {
  channel: LGAChannel
  messages: LGAMessage[]
  count: number
}

export function useLGAChannel() {
  return useQuery({
    queryKey: ['lga-channel'],
    queryFn: () => api.get<ChannelResponse>('/lga/channel'),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
}

export interface PostLGAInput {
  category: LGAMessage['category']
  priority: LGAMessage['priority']
  title: string
  body: string
  expiresInHours?: number
}

export function useLGAChannelActions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: PostLGAInput) => api.post<{ message: LGAMessage }>('/lga/channel', input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['lga-channel'] })
    },
  })
}
