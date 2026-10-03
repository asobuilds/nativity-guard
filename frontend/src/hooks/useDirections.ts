import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface RouteStep {
  instruction: string
  distanceMeters: number
}

export interface Route {
  distanceMeters: number
  durationSeconds: number
  geometry: string // encoded polyline
  steps: RouteStep[]
  summary: string
  profile: 'driving' | 'cycling' | 'walking'
}

interface DirectionsResponse {
  route: Route
}

interface Point {
  lat: number
  lng: number
}

/**
 * A route between two points. Enabled only when both points are known.
 * Cached for 5 minutes — routes change only when a user picks a new
 * destination.
 */
export function useDirections(
  from: Point | null,
  to: Point | null,
  profile: 'driving' | 'cycling' | 'walking' = 'driving',
) {
  const enabled = Boolean(from && to)

  return useQuery({
    queryKey: [
      'directions',
      from?.lat.toFixed(5),
      from?.lng.toFixed(5),
      to?.lat.toFixed(5),
      to?.lng.toFixed(5),
      profile,
    ],
    queryFn: () =>
      api.post<DirectionsResponse>('/directions', {
        fromLat: from!.lat,
        fromLng: from!.lng,
        toLat: to!.lat,
        toLng: to!.lng,
        profile,
      }),
    select: (data) => data.route,
    enabled,
    staleTime: 5 * 60_000,
    retry: 1,
  })
}