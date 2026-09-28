import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { Role } from '@/types/api'

export interface SystemStats {
  totalUsers: number
  totalCases: number
  totalUnits: number
  totalOfficers: number
  totalSOS: number
  totalSuspects: number
  dailyActive: number
  pendingCases: number
  resolvedCases: number
}

export interface AuditLogEntry {
  id: string
  userId: string
  action: string
  entityType: string
  entityId: string
  oldValue: string
  newValue: string
  ipAddress: string
  userAgent: string
  timestamp: string
  createdAt: string
  updatedAt: string
  user?: { id: string; firstName: string; lastName: string; email: string }
}

export interface ActivityLogEntry {
  id: string
  userId: string
  sessionId: string
  activityType: string
  description: string
  ipAddress: string
  device: string
  location: string
  duration: number
  createdAt: string
  user?: { id: string; firstName: string; lastName: string; email: string }
}

export interface SystemHealth {
  id: string
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  activeUsers: number
  totalRequests: number
  responseTime: number
  databaseStatus: string
  serverStatus: string
  uptime: number
  lastCheck: string
  createdAt: string
  updatedAt: string
}

export interface SuperAdminUser {
  id: string
  email: string
  phone: string
  firstName: string
  lastName: string
  role: Role
  status: string
  createdAt: string
  lastLogin: string | null
  unitId: string | null
}

export const superAdminKeys = {
  stats: ['super', 'stats'] as const,
  users: ['super', 'users'] as const,
  auditLogs: ['super', 'audit', 'logs'] as const,
  activities: ['super', 'audit', 'activities'] as const,
  health: ['super', 'audit', 'health'] as const,
}

export function useSystemStats() {
  return useQuery({
    queryKey: superAdminKeys.stats,
    queryFn: () => api.get<SystemStats>('/admin/stats'),
    staleTime: 60_000,
  })
}

export function useAllUsers() {
  return useQuery({
    queryKey: superAdminKeys.users,
    queryFn: () => api.get<{ users: SuperAdminUser[]; total: number }>('/admin/users'),
    staleTime: 60_000,
  })
}

export function useAuditLogs() {
  return useQuery({
    queryKey: superAdminKeys.auditLogs,
    queryFn: () => api.get<{ auditLogs: AuditLogEntry[] }>('/audit/logs'),
    select: (d) => d.auditLogs,
    staleTime: 30_000,
  })
}

export function useActivityLogs() {
  return useQuery({
    queryKey: superAdminKeys.activities,
    queryFn: () => api.get<{ activities: ActivityLogEntry[] }>('/audit/activities'),
    select: (d) => d.activities,
    staleTime: 30_000,
  })
}

export function useSystemHealth() {
  return useQuery({
    queryKey: superAdminKeys.health,
    queryFn: () => api.get<{ health: SystemHealth }>('/audit/health'),
    select: (d) => d.health,
    staleTime: 30_000,
  })
}

export function useSuspendUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => api.post<{ message: string }>('/admin/users/' + userId + '/suspend'),
    onSuccess: () => qc.invalidateQueries({ queryKey: superAdminKeys.users }),
  })
}

export function useActivateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => api.post<{ message: string }>('/admin/users/' + userId + '/activate'),
    onSuccess: () => qc.invalidateQueries({ queryKey: superAdminKeys.users }),
  })
}

export function useUpdateUserRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { userId: string; role: Role }) =>
      api.put<{ message: string }>('/admin/users/' + input.userId + '/role', { role: input.role }),
    onSuccess: () => qc.invalidateQueries({ queryKey: superAdminKeys.users }),
  })
}