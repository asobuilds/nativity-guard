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
  /** Head admins only may post (decision D-015); everyone eligible may read. */
  canPost: boolean
}

export function useLGAChannel() {
  return useQuery({
    queryKey: ['lga-channel'],
    queryFn: () => api.get<ChannelResponse>('/lga/channel'),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
}

export interface LGAGroupMember {
  id: string
  unitId: string
  unitName: string
  joinedAt: string
}

export interface LGAGroupState {
  group: { state: string; lga: string }
  members: LGAGroupMember[]
  count: number
  /** Whether the caller's own unit has joined the convergence group. */
  joined: boolean
  canPost: boolean
}

/** The LGA convergence group roster (RES-012). */
export function useLGAGroup() {
  return useQuery({
    queryKey: ['lga-group'],
    queryFn: () => api.get<LGAGroupState>('/community/lga-group/members'),
    staleTime: 30_000,
  })
}

export function useLGAGroupActions() {
  const qc = useQueryClient()
  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['lga-group'] })
    void qc.invalidateQueries({ queryKey: ['lga-channel'] })
  }
  const join = useMutation({
    mutationFn: () => api.post('/community/lga-group/join'),
    onSuccess: invalidate,
  })
  const leave = useMutation({
    mutationFn: () => api.post('/community/lga-group/leave'),
    onSuccess: invalidate,
  })
  return { join, leave }
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
