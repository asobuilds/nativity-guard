import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type {
  TaskForceCreateResponse,
  TaskForceDetailResponse,
  TaskForceEnlistResponse,
  TaskForceInviteResponse,
  TaskForceListResponse,
  UnitOfficersResponse,
} from '@/types/api'

export const taskForceKeys = {
  all: ['taskforce'] as const,
  list: () => ['taskforce', 'list'] as const,
  detail: (id: string) => ['taskforce', 'detail', id] as const,
  officers: (id: string) => ['taskforce', 'officers', id] as const,
}

export function useMyTaskForces() {
  return useQuery({
    queryKey: taskForceKeys.list(),
    queryFn: () => api.get<TaskForceListResponse>('/taskforce'),
    select: (data) => data.taskForces,
  })
}

export function useTaskForce(id: string | undefined) {
  return useQuery({
    queryKey: taskForceKeys.detail(id ?? ''),
    queryFn: () => api.get<TaskForceDetailResponse>(`/taskforce/${id}`),
    enabled: Boolean(id),
  })
}

export function useTaskForceOfficers(id: string | undefined, enabled = true) {
  return useQuery({
    queryKey: taskForceKeys.officers(id ?? ''),
    queryFn: () => api.get<UnitOfficersResponse>(`/taskforce/${id}/officers`),
    select: (data) => data.officers,
    enabled: Boolean(id) && enabled,
    retry: false,
  })
}

function useInvalidateTaskForce(id?: string) {
  const queryClient = useQueryClient()
  return async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: taskForceKeys.list() }),
      id ? queryClient.invalidateQueries({ queryKey: taskForceKeys.detail(id) }) : Promise.resolve(),
      id ? queryClient.invalidateQueries({ queryKey: taskForceKeys.officers(id) }) : Promise.resolve(),
    ])
  }
}

export function useCreateTaskForce() {
  const invalidate = useInvalidateTaskForce()
  return useMutation({
    mutationFn: (input: { name: string; hostUnitId: string }) =>
      api.post<TaskForceCreateResponse>('/taskforce', input),
    onSuccess: () => void invalidate(),
  })
}

export function useInviteTaskForceUnit(id: string | undefined) {
  const invalidate = useInvalidateTaskForce(id)
  return useMutation({
    mutationFn: (input: { unitId: string; expiresInHours?: number }) =>
      api.post<TaskForceInviteResponse>(`/taskforce/${id}/invite`, input),
    onSuccess: () => void invalidate(),
  })
}

export function useJoinTaskForce(id: string | undefined) {
  const invalidate = useInvalidateTaskForce(id)
  return useMutation({
    mutationFn: (input: { code: string }) =>
      api.post<{ message: string }>(`/taskforce/${id}/join`, input),
    onSuccess: () => void invalidate(),
  })
}

export function useEnlistTaskForceOfficers(id: string | undefined) {
  const invalidate = useInvalidateTaskForce(id)
  return useMutation({
    mutationFn: (officerIds: string[]) =>
      api.post<TaskForceEnlistResponse>(`/taskforce/${id}/enlist`, { officerIds }),
    onSuccess: () => void invalidate(),
  })
}

export function useDisbandTaskForce(id: string | undefined) {
  const invalidate = useInvalidateTaskForce(id)
  return useMutation({
    mutationFn: () => api.post<{ message: string }>(`/taskforce/${id}/disband`),
    onSuccess: () => void invalidate(),
  })
}

export function useLeaveTaskForce(id: string | undefined) {
  const invalidate = useInvalidateTaskForce(id)
  return useMutation({
    mutationFn: (unitId?: string) =>
      api.post<{ message: string }>(`/taskforce/${id}/leave`, unitId ? { unitId } : {}),
    onSuccess: () => void invalidate(),
  })
}
