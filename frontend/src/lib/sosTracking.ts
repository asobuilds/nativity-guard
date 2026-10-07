import type { SosAlert } from '@/types/api'

export interface SosResponder {
  id: string
  unitId: string
  role: 'primary' | 'support'
  acceptedAt: string
  assigned?: boolean
  assignedUserId?: string
  isMyAssignment?: boolean
  canManage?: boolean
  unit?: { id: string; name: string; type?: string }
}

export interface SosDetail extends SosAlert {
  severity?: 'normal' | 'critical'
  dispatchState?: 'open' | 'locked' | 'multi_responder' | 'resolved'
  acceptedByUnitId?: string
  acceptedAt?: string
  jurisdictionState?: string
  jurisdictionLga?: string
  responders: SosResponder[]
}

export interface ResponderLocation {
  id: string
  unitId: string
  role: string
  latitude: number
  longitude: number
  accuracy: number
  recordedAt: string
}

export interface SosDetailResponse {
  sos?: SosDetail
  alert?: SosDetail
  responders?: SosResponder[]
}

export function normaliseSosDetail(data: SosDetailResponse): SosDetail {
  const sos = data.sos ?? data.alert
  if (!sos?.id) throw new Error('The server did not return an SOS alert.')
  return { ...sos, responders: data.responders ?? sos.responders ?? [] }
}

export function isSosActive(sos: SosDetail): boolean {
  return sos.status !== 'resolved' && sos.status !== 'cancelled' && sos.dispatchState !== 'resolved'
}

export function freshResponderLocations(rows: ResponderLocation[], now: number): ResponderLocation[] {
  return rows.filter((r) => {
    const age = now - Date.parse(r.recordedAt)
    return Number.isFinite(age) && age >= -10_000 && age <= 120_000 &&
      Number.isFinite(r.latitude) && Math.abs(r.latitude) <= 90 &&
      Number.isFinite(r.longitude) && Math.abs(r.longitude) <= 180 &&
      Number.isFinite(r.accuracy) && r.accuracy >= 0
  })
}
