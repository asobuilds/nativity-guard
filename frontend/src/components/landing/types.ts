import type { CaseStatus, PriorityLevel } from '@/types/api'

/** Public case representation returned by GET /api/v1/public/cases */
export interface PublicCase {
  id: string
  unitId: string
  title: string
  description: string
  location?: string
  latitude: number
  longitude: number
  gisLatitude?: number
  gisLongitude?: number
  status: CaseStatus
  priorityLevel: PriorityLevel
  trackingId: string
  isPublic: boolean
  isAnonymous?: boolean
  createdAt: string
  updatedAt: string
}

/** Leaderboard row returned by GET /api/v1/public/leaderboard */
export interface LeaderboardRow {
  unitId: string
  compositeScore: number
  tier: string
  officerCount: number
  directCount: number
  unitName?: string
}

/** Blueprint description returned by GET /api/v1/public/blueprint */
export interface PlatformBlueprint {
  name: string
  tagline: string
  purpose: string
  principles: Array<{ title: string; body: string }>
  roles: Array<{ name: string; capabilities: string[] }>
  architecture: {
    backend: string
    frontend: string
    infra: string
  }
  caseLifecycle: string[]
  security: string[]
  governance: {
    elections: {
      term_months: number
      staggered_rotation: boolean
      stagger_months: number
      term_limit: number
      cooling_off_months: number
      seat_bands: Record<string, number>
      quorum_percent: number
    }
    revocation: {
      regular_admin: string
      head_admin: string
      one_vote_per: string
    }
  }
  contact: {
    support_email: string
    platform_url: string
  }
}

/** Platform supporter returned by GET /api/v1/public/platform/supporters */
export interface PublicSupporter {
  donorName: string
  amount: number
  message?: string
  createdAt: string
}

/** Supported official action types that require authentication */
export type OfficialActionType =
  | 'sos_broadcast'
  | 'report_incident'
  | 'join_unit'
  | 'enlist_officer'
  | 'vote_election'
  | 'vote_revocation'
  | 'community_post'
  | 'community_reply'
  | 'rate_officer'
  | 'follow_case'
  | 'ai_consultation'
  | 'subscribe_alerts'

/** Action context passed when opening the ActionAuthModal */
export interface ActionContext {
  type: OfficialActionType
  title: string
  description: string
  badgeText?: string
  badgeVariant?: 'emergency' | 'signal' | 'warn' | 'ok'
  returnUrl?: string
  payload?: Record<string, unknown>
}

/** Serialized pending action stored during login/signup */
export interface StoredPendingAction {
  context: ActionContext
  savedAt: number
}
