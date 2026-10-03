import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export type POICategory =
  | 'hospital'
  | 'police'
  | 'fire'
  | 'pharmacy'
  | 'bank'
  | 'school'
  | 'church'
  | 'mosque'
  | 'landmark'

export interface POI {
  id: string
  category: POICategory
  name: string
  latitude: number
  longitude: number
  address?: string
  phone?: string
}

interface POIResponse {
  items: POI[]
  count: number
  radius: number
  location: { latitude: number; longitude: number }
}

/**
 * Points of interest around a coordinate. The backend caches each rounded
 * coordinate + category set for 1 hour, so switching filters is cheap.
 *
 * Pass `categories: []` to disable the request — useful when the user has
 * turned POI display off.
 */
export function useMapPOIs(
  latitude: number | null | undefined,
  longitude: number | null | undefined,
  radiusMeters: number,
  categories: POICategory[],
) {
  const hasCoords =
    typeof latitude === 'number' &&
    typeof longitude === 'number' &&
    latitude !== 0 &&
    longitude !== 0

  const catKey = [...categories].sort().join(',')

  return useQuery({
    queryKey: ['map-pois', latitude?.toFixed(3), longitude?.toFixed(3), radiusMeters, catKey],
    queryFn: () => {
      const qs = new URLSearchParams({
        lat: String(latitude),
        lng: String(longitude),
        radius: String(radiusMeters),
        categories: categories.join(','),
      })
      return api.get<POIResponse>(`/map/pois?${qs.toString()}`)
    },
    enabled: hasCoords && categories.length > 0,
    staleTime: 30 * 60_000, // 30 min — backend caches for 1 hour
    refetchOnWindowFocus: false,
    retry: 1,
  })
}

export const POI_META: Record<POICategory, { label: string; color: string }> = {
  hospital: { label: 'Hospital', color: '#ef4444' },
  police: { label: 'Police', color: '#3b82f6' },
  fire: { label: 'Fire station', color: '#f97316' },
  pharmacy: { label: 'Pharmacy', color: '#22c55e' },
  bank: { label: 'Bank', color: '#eab308' },
  school: { label: 'School', color: '#8b5cf6' },
  church: { label: 'Church', color: '#a855f7' },
  mosque: { label: 'Mosque', color: '#14b8a6' },
  landmark: { label: 'Landmark', color: '#f59e0b' },
}