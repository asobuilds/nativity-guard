import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { PlatformBlueprint } from '@/components/landing/types'

export const publicBlueprintKeys = {
  all: ['public-blueprint'] as const,
}

/**
 * Public policy and architecture claims must come from the actual backend.
 * Never invent a replacement blueprint if the API is offline or incomplete.
 */
export function usePublicBlueprint() {
  return useQuery({
    queryKey: publicBlueprintKeys.all,
    queryFn: async () => {
      const res = await api.get<PlatformBlueprint>('/public/blueprint')
      if (!res || typeof res.name !== 'string' || !res.governance?.elections) {
        throw new Error('Public governance information is unavailable.')
      }
      return res
    },
    staleTime: 60 * 60_000,
    networkMode: 'offlineFirst',
  })
}
