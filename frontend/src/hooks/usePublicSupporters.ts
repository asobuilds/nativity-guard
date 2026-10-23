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

/** Empty and unavailable results are not replaced with fictional donations. */
export function usePublicSupporters() {
  return useQuery({
    queryKey: publicSupporterKeys.supporters,
    queryFn: async () => {
      const res = await api.get<{ supporters: PublicSupporter[] }>('/public/platform/supporters')
      return res.supporters ?? []
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

