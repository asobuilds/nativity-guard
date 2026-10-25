import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface UnitLandmark {
  id: string
  unitId: string
  name: string
  type: 'Safe House' | 'Known Hotspot' | 'Checkpoint' | 'Post' | string
  latitude: number
  longitude: number
  createdAt: string
}

export function useUnitLandmarks(unitId?: string | null) {
  return useQuery({
    queryKey: ['unit-landmarks', unitId],
    queryFn: () => api.get<{ landmarks: UnitLandmark[] }>(`/units/${unitId}/landmarks`),
    select: (d) => d.landmarks,
    enabled: Boolean(unitId),
  })
}

export function useCreateUnitLandmark(unitId?: string | null) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { name: string; type: string; latitude: number; longitude: number }) =>
      api.post(`/units/${unitId}/landmarks`, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['unit-landmarks', unitId] })
    },
  })
}

export function useDeleteUnitLandmark(unitId?: string | null) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (landmarkId: string) => api.delete(`/units/${unitId}/landmarks/${landmarkId}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['unit-landmarks', unitId] })
    },
  })
}
