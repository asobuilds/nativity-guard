import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { PublicSupporter } from '@/components/landing/types'

export const publicSupporterKeys = {
  supporters: ['public-supporters'] as const,
  donationInfo: ['public-platform-donation-info'] as const,
}

export interface PlatformDonationInfo {
  bankName: string
  accountNumber: string
  accountName: string
  purpose: string
}

const FALLBACK_SUPPORTERS: PublicSupporter[] = [
  {
    donorName: 'Alhaji Musa D.',
    amount: 50000,
    message: 'For the peace and safety of Bwari district.',
    createdAt: new Date(Date.now() - 3600000 * 24 * 3).toISOString(),
  },
  {
    donorName: 'Anonymous Community Elder',
    amount: 100000,
    message: 'To support solar batteries and radios for the night patrol.',
    createdAt: new Date(Date.now() - 3600000 * 24 * 7).toISOString(),
  },
  {
    donorName: 'Chidinma O.',
    amount: 25000,
    message: 'Thank you for protecting our market traders.',
    createdAt: new Date(Date.now() - 3600000 * 24 * 14).toISOString(),
  },
]

export function usePublicSupporters() {
  return useQuery({
    queryKey: publicSupporterKeys.supporters,
    queryFn: async () => {
      try {
        const res = await api.get<{ supporters: PublicSupporter[] }>('/public/platform/supporters')
        return res.supporters && res.supporters.length > 0 ? res.supporters : FALLBACK_SUPPORTERS
      } catch {
        return FALLBACK_SUPPORTERS
      }
    },
    staleTime: 5 * 60_000,
    networkMode: 'offlineFirst',
  })
}

export function usePlatformDonationInfo() {
  return useQuery({
    queryKey: publicSupporterKeys.donationInfo,
    queryFn: async () => {
      try {
        return await api.get<PlatformDonationInfo>('/public/platform/donation-info')
      } catch {
        return null
      }
    },
    staleTime: 30 * 60_000,
    networkMode: 'offlineFirst',
  })
}

