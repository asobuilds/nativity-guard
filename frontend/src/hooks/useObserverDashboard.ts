import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { ObserverIncidentsResponse, ObserverUnitsResponse } from '@/types/api'

export const observerQueryKeys = {
  all: ['observer'] as const,
  incidents: (limit: number) => ['observer', 'incidents', limit] as const,
  units: ['observer', 'units'] as const,
}

export function observerIncidentsPath(limit = 50): string {
  return `/observer/incidents?limit=${limit}`
}

export const OBSERVER_UNITS_PATH = '/observer/units'

export function useObserverIncidents(limit = 50) {
  return useQuery({
    queryKey: observerQueryKeys.incidents(limit),
    queryFn: () => api.get<ObserverIncidentsResponse>(observerIncidentsPath(limit)),
    staleTime: 30_000,
  })
}

export function useObserverUnits() {
  return useQuery({
    queryKey: observerQueryKeys.units,
    queryFn: () => api.get<ObserverUnitsResponse>(OBSERVER_UNITS_PATH),
    staleTime: 30_000,
  })
}
