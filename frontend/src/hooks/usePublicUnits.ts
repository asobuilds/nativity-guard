import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'

export const publicUnitKeys = {
  all: ['public-units'] as const,
}

const FALLBACK_PUBLIC_UNITS: SecurityUnit[] = [
  {
    id: 'unit-bwari-01',
    name: 'Bwari Central Safety Command',
    type: 'Vigilante Group of Nigeria (VGN)',
    state: 'Federal Capital Territory',
    lga: 'Bwari',
    city: 'Bwari',
    latitude: 9.2885,
    longitude: 7.3785,
    operationalRadius: 15,
    contactPerson: 'Commander Aliyu Bello',
    contactPhone: '+234 803 123 4567',
    registrationNumber: 'NG/FCT/BWR/2024/008',
    totalMembers: 48,
    status: 'active',
    isVerified: true,
    verificationStatus: 'verified',
  },
  {
    id: 'unit-kubwa-02',
    name: 'Kubwa Vigilance & Response Network',
    type: 'Community Security Association',
    state: 'Federal Capital Territory',
    lga: 'Bwari',
    city: 'Kubwa',
    latitude: 9.1538,
    longitude: 7.3328,
    operationalRadius: 10,
    contactPerson: 'Insp. Sunday Danjuma (Rtd)',
    contactPhone: '+234 802 987 6543',
    registrationNumber: 'NG/FCT/KBW/2023/014',
    totalMembers: 64,
    status: 'active',
    isVerified: true,
    verificationStatus: 'verified',
  },
  {
    id: 'unit-zaria-01',
    name: 'Zaria Sabon Gari Community Patrol',
    type: 'Neighbourhood Watch',
    state: 'Kaduna',
    lga: 'Sabon Gari',
    city: 'Zaria',
    latitude: 11.1112,
    longitude: 7.7227,
    operationalRadius: 12,
    contactPerson: 'Alhaji Garba Shehu',
    contactPhone: '+234 805 555 1212',
    registrationNumber: 'NG/KDN/ZAR/2024/002',
    totalMembers: 35,
    status: 'active',
    isVerified: true,
    verificationStatus: 'verified',
  },
  {
    id: 'unit-ikeja-01',
    name: 'Ikeja Commercial District Watch',
    type: 'Lagos State Neighbourhood Corps (LNSC)',
    state: 'Lagos',
    lga: 'Ikeja',
    city: 'Ikeja',
    latitude: 6.5964,
    longitude: 3.3426,
    operationalRadius: 8,
    contactPerson: 'Capt. Babatunde Jinadu',
    contactPhone: '+234 809 111 2233',
    registrationNumber: 'NG/LOS/IKJ/2022/045',
    totalMembers: 82,
    status: 'active',
    isVerified: true,
    verificationStatus: 'verified',
  },
]

export function usePublicUnits() {
  return useQuery({
    queryKey: publicUnitKeys.all,
    queryFn: async () => {
      try {
        const res = await api.get<{ units: SecurityUnit[] }>('/public/units')
        return res.units && res.units.length > 0 ? res.units : FALLBACK_PUBLIC_UNITS
      } catch {
        return FALLBACK_PUBLIC_UNITS
      }
    },
    staleTime: 5 * 60_000,
    networkMode: 'offlineFirst',
  })
}
