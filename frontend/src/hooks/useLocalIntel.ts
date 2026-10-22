import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface LocalIntelFact {
  id: string
  fact: string
  category: string
  confidence: string
  state?: string
  lga?: string
  upvotes: number
  hasUpvoted: boolean
  expiresAt?: string
  createdAt: string
}

interface LocalIntelResponse {
  items: LocalIntelFact[]
}

interface UpvoteResponse {
  id: string
  upvotes: number
  hasUpvoted: boolean
}

/**
 * Unexpired facts for the caller's area, most-upvoted first. The backend
 * already drops expired rows, so the client never needs to filter by time.
 */
export function useLocalIntel(limit = 5) {
  return useQuery({
    queryKey: ['local-intel'],
    queryFn: () => api.get<LocalIntelResponse>('/local-intel'),
    select: (data) => data.items.slice(0, limit),
    staleTime: 60_000,
    retry: 1,
  })
}

export function useUpvoteLocalIntel() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post<UpvoteResponse>(`/local-intel/${id}/upvote`, {}),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['local-intel'] })
    },
  })
}
