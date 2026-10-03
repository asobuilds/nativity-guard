import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

interface PendingCase {
  id: string
  title: string
  trackingId: string
  status: string
  priorityLevel: string
  location: string
  createdAt: string
}

interface SosAlert {
  id: string
  description?: string
  status: string
  priority: string
  latitude: number
  longitude: number
  createdAt: string
}

interface Application {
  membershipId: string
  userId: string
  firstName: string
  lastName: string
  email: string
  avatarPath?: string
  role: string
  createdAt: string
}

export interface UnitInbox {
  pendingCases: PendingCase[]
  pendingCaseCount: number
  sosAlerts: SosAlert[]
  sosAlertCount: number
  applications: Application[]
  applicationCount: number
}

export function useUnitInbox(unitId: string | undefined) {
  return useQuery({
    queryKey: ['unit-inbox', unitId],
    queryFn: () => api.get<UnitInbox>(`/units/${unitId}/inbox`),
    enabled: Boolean(unitId),
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}
