import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { PublicCase } from '@/components/landing/types'

export const publicCaseKeys = {
  all: ['public-cases'] as const,
}

const FALLBACK_PUBLIC_CASES: PublicCase[] = [
  {
    id: 'sample-case-1',
    unitId: 'unit-bwari-01',
    title: 'Perimeter fence breach and equipment vandalism',
    description: 'Reported unauthorized entry into the community storage yard overnight. Unit dispatched.',
    location: 'Bwari Central, Abuja FCT',
    latitude: 9.2885,
    longitude: 7.3785,
    status: 'investigating',
    priorityLevel: 'P2',
    trackingId: 'NG-2026-00194821',
    isPublic: true,
    isAnonymous: false,
    createdAt: new Date(Date.now() - 3600000 * 4).toISOString(),
    updatedAt: new Date(Date.now() - 3600000 * 2).toISOString(),
  },
  {
    id: 'sample-case-2',
    unitId: 'unit-kubwa-02',
    title: 'Distress call: suspicious night prowler on Sector 4 lane',
    description: 'Resident observed two individuals testing gate locks along Sector 4. Patrol unit arrived on scene.',
    location: 'Kubwa Phase 3, Abuja FCT',
    latitude: 9.1538,
    longitude: 7.3328,
    status: 'on_scene',
    priorityLevel: 'P1',
    trackingId: 'NG-2026-00194822',
    isPublic: true,
    isAnonymous: true,
    createdAt: new Date(Date.now() - 3600000 * 6).toISOString(),
    updatedAt: new Date(Date.now() - 3600000 * 1).toISOString(),
  },
  {
    id: 'sample-case-3',
    unitId: 'unit-zaria-01',
    title: 'Market stall inventory dispute mediation',
    description: 'Commercial boundary dispute resolved peacefully with community peace committee oversight.',
    location: 'Sabon Gari Market, Zaria, Kaduna',
    latitude: 11.1112,
    longitude: 7.7227,
    status: 'closed',
    priorityLevel: 'P3',
    trackingId: 'NG-2026-00194823',
    isPublic: true,
    isAnonymous: false,
    createdAt: new Date(Date.now() - 3600000 * 28).toISOString(),
    updatedAt: new Date(Date.now() - 3600000 * 12).toISOString(),
  },
]

export function usePublicCases() {
  return useQuery({
    queryKey: publicCaseKeys.all,
    queryFn: async () => {
      try {
        const res = await api.get<{ cases: PublicCase[] }>('/public/cases')
        return res.cases && res.cases.length > 0 ? res.cases : FALLBACK_PUBLIC_CASES
      } catch {
        return FALLBACK_PUBLIC_CASES
      }
    },
    staleTime: 60_000,
    networkMode: 'offlineFirst',
  })
}

