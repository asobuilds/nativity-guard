/**
 * Nativity Guard API types.
 *
 * Hand-written from the verified Go contract in backend/ (handlers + models).
 * Envelopes are intentional: the API wraps collections, e.g. `{ cases: [...] }`.
 * Keep this file in sync with the backend — it is the frontend's contract.
 */

export type Role = 'citizen' | 'officer' | 'unit_admin' | 'super_admin'

/**
 * The eight states the backend case workflow actually sets.
 *
 * Declared as one const object so the literals live in exactly one place — a
 * contract correction is then a single edit rather than a hunt. They are
 * snake_case because the backend stores plain strings: `case_workflow_handler.go`
 * compares against `"pending"` / `"on_scene"` / … directly, and
 * `case_review_handler.go` uses `models.CaseStatus*` constants of the same shape.
 *
 * NOTE: the three review-phase literals are the ones frontReadme.md §0.4 flags as
 * needing confirmation against `backend/models` — see the verification command there.
 */
export const CASE_STATUS = {
  pending: 'pending',
  assigned: 'assigned',
  dispatched: 'dispatched',
  onScene: 'on_scene',
  investigating: 'investigating',
  pendingAdminReview: 'pending_admin_review',
  adminChangesRequested: 'admin_changes_requested',
  closed: 'closed',
} as const

export type CaseStatus = (typeof CASE_STATUS)[keyof typeof CASE_STATUS]

/** Backend priority banding (see CreateCase in handlers/case_handler.go). */
export type PriorityLevel = 'P1' | 'P2' | 'P3'

export interface User {
  id: string
  unitId?: string | null
  email: string
  phone?: string
  photoUrl?: string
  avatarPath?: string
  coverPath?: string
  firstName: string
  lastName: string
  role: Role
  status?: string
  createdAt?: string
  updatedAt?: string
}

export interface Evidence {
  id: string
  caseId: string
  uploadedBy: string
  /** Free-form today (image | video | audio | document …). */
  type: string
  fileUrl: string
  description?: string
  latitude?: number
  longitude?: number
  isVerified: boolean
  uploadedAt: string
  createdAt: string
}

export interface Progress {
  id: string
  caseId: string
  officerId: string
  action: string
  description?: string
  createdAt: string
}

export interface CaseTimelineEntry {
  id: string
  caseId: string
  userId: string
  action: string
  description?: string
  status?: string
  metadata?: string
  createdAt: string
  updatedAt?: string
  user?: User
}

export interface CaseOfficer {
  id: string
  caseId: string
  officerId: string
  /** primary | investigator | support */
  role: string
  createdAt?: string
}

export interface CaseFeedback {
  id: string
  caseId: string
  userId: string
  rating: number
  comment?: string
  categories?: string
  isPublic?: boolean
  createdAt?: string
}

/** SOS contract used by the emergency console; live response shape needs verification. */
export interface SosAlert {
  id: string
  userId: string

  status: 'pending' | 'dispatched' | 'resolved' | 'escalated'
  latitude: number
  longitude: number
  priority: 'high' | 'critical'
  locationGeohash?: string
  isAnonymous?: boolean
  description?: string
  unitId?: string
  emergencyContacts?: string[]
  medicalInfo?: string
  createdAt: string
}

export interface SendSosInput {
  latitude: number
  longitude: number
  priority: SosAlert['priority']
  unitId?: string
  emergencyContacts?: string[]
  medicalInfo?: string
  /**
   * Opt out of exposing the reporter's identity on the public feed. Responders
   * still see the precise coordinates; the toggle only strips identifying
   * context from displays the public can reach (see `POST /sos`).
   */
  hideLocation?: boolean
}

export interface Case {
  id: string
  unitId: string
  reportedBy: string
  assignedTo?: string | null
  title: string
  description: string
  incidentDate?: string
  location?: string
  latitude: number
  longitude: number
  status: CaseStatus
  priority?: string
  priorityLevel: PriorityLevel
  trackingId: string
  gisLatitude: number
  gisLongitude: number
  transferDetails?: string
  isPublic: boolean
  assignedAt?: string | null
  dispatchedAt?: string | null
  arrivedAt?: string | null
  closedAt?: string | null
  closedBy?: string | null
  approvedBy?: string | null
  finalReport?: string
  createdAt: string
  updatedAt: string
  evidence?: Evidence[]
  progress?: Progress[]
}

/**
 * One decision recorded against a case's closure review (`models.CaseReview`).
 *
 * `decision` stays a plain string on purpose, even though the backend's
 * `models.CaseReviewDecision*` literals are now known — `approve`,
 * `request_changes` and `deescalate`. Only the first two are reachable from this
 * frontend, so a narrow union would buy nothing today and would make a decision
 * added later render as a `never`-typed crash instead of as itself. Unknown
 * values fall through `humanizeDecision` to Title Case.
 */
export interface CaseReview {
  id: string
  caseId: string
  adminId: string
  /** Verified against `models.CaseReviewDecision*`: `approve` | `request_changes` | `deescalate`. */
  decision: string
  comment: string
  createdAt: string
}

/**
 * An officer's narrative for one reporting week (`models.CaseWeeklyUpdate`).
 *
 * The week is Monday 00:00 UTC → the following Monday, computed **server-side**;
 * `weekStart` is the key the backend deduplicates on, so a second submission for
 * the same week is refused rather than merged.
 */
export interface CaseWeeklyUpdate {
  id: string
  caseId: string
  officerId: string
  weekStart: string
  weekEnd: string
  summary: string
  investigation: string
  actionsTaken?: string
  findings?: string
  evidenceSummary?: string
  outstandingActions?: string
  nextSteps?: string
  /**
   * Whether this update is exposed to the case reporter. Backed by
   * `CitizenVisible bool ... json:"citizenVisible"` in `models.CaseWeeklyUpdate.go`
   * — verified 2026-09-14, and without `omitempty`, so it is always present.
   *
   * The server does the filtering, so this is a **label, not a permission check**:
   * if it is `false`, say nothing about visibility rather than announcing a privacy
   * guarantee the client cannot enforce.
   */
  citizenVisible: boolean
  submittedAt?: string | null
  createdAt: string
}

/** Body of `POST /cases/:id/weekly-update` — the first two are required. */
export interface CaseWeeklyUpdateInput {
  summary: string
  investigation: string
  actionsTaken?: string
  findings?: string
  evidenceSummary?: string
  outstandingActions?: string
  nextSteps?: string
}

export interface SecurityUnit {
  id: string
  name: string
  type: string
  latitude: number
  longitude: number
  /** Operational coverage radius in kilometres. */
  operationalRadius: number
  state?: string
  lga?: string
  city?: string
  coverageArea?: string
  contactPerson?: string
  contactPhone?: string
  contactEmail?: string
  registrationNumber?: string
  /** Local government ward. Omitted by the API when unset (`omitempty`). */
  ward?: string
  /** `YYYY-MM-DD`. Omitted by the API when unset (`omitempty`). */
  formationDate?: string
  totalMembers?: number
  brandName?: string
  brandLogoUrl?: string
  brandCoverUrl?: string
  memberCount?: number
  adminCount?: number
  status: string
  isVerified: boolean
  verificationStatus: string

  // Section B — commander details (from the registration form)
  commanderName?: string
  commanderNin?: string
  commanderPhoneAlt?: string
  commanderOccupation?: string
  commanderPriorExperience?: string
  // Section C — operational & equipment profile
  hasUniform?: boolean
  uniformDescription?: string
  shiftPattern?: string
  permittedTools?: string
  weaponsRegistered?: boolean
  // Section D — traditional & local endorsement
  kindredHeadName?: string
  kindredHeadPhone?: string
  wardHeadName?: string
  wardHeadPhone?: string
}

/** Enriched unit returned by GET /units/nearby. */
export interface UnitWithDistance extends SecurityUnit {
  distance: number
  isInRange: boolean
}

/**
 * A roster officer — the backend's `officers` table, which is a *different*
 * entity from the login `User`.
 *
 * This matters for assignment: `POST /cases/:id/assign` takes an `officers` id,
 * NOT a user id, and the backend rejects any officer whose unit differs from the
 * case's unit (`OfficerID` is checked against `Officer.UnitID`).
 */
export interface UnitOfficer {
  id: string
  unitId: string
  name: string
  rank: string
  badgeNumber: string
  /** Duty role: patrol, investigator, commander, dispatch … */
  role: string
  phone?: string
  email?: string
  joinedDate?: string
  status: string
}

export interface Notification {
  id: string
  userId: string
  title: string
  message: string
  type?: string
  status: 'unread' | 'read' | string
  createdAt: string
}

/* ------------------------------------------------------------------ */
/* Response envelopes                                                  */
/* ------------------------------------------------------------------ */

export interface LoginResponse {
  token: string
  /** Required: the backend 500s rather than omit it. See `RefreshResponse`. */
  refreshToken: string
  user: Pick<User, 'id' | 'email' | 'firstName' | 'lastName' | 'role'>
}

/**
 * The body of `POST /auth/refresh`.
 *
 * Refresh **rotates**: the token you send is spent, and the response carries its
 * replacement. A caller that keeps only `token` will find its next refresh
 * rejected as revoked — so `refreshToken` is not optional here.
 */
export interface RefreshResponse {
  token: string
  refreshToken: string
}

/**
 * The body of `POST /auth/register`, which returns **201 and no token**.
 *
 * Registration deliberately does not sign you in: the response carries a message
 * and the created user, nothing more. A caller that expected a token here would
 * fail silently, so the shape is spelled out rather than inferred from
 * `LoginResponse`. `role` is always `"citizen"` — the handler hardcodes it and
 * ignores any `role` sent in the request.
 */
export interface RegisterResponse {
  message: string
  user: Pick<User, 'id' | 'email' | 'firstName' | 'lastName' | 'role'>
}

/**
 * The body of `POST /auth/forgot-password`.
 *
 * Always 200, whether or not the identifier matches an account: the endpoint is
 * deliberately non-enumerating and its message says so. That message mentions a
 * "reset link", but what actually arrives is a 6-digit code — so the screens show
 * their own copy rather than echoing this string back at someone who is waiting
 * for a link.
 */
export interface ForgotPasswordResponse {
  message: string
}

/**
 * The body of `POST /auth/reset-password`.
 *
 * Success is not local to this device: the service revokes every session and
 * refresh token the account holds, so any session open in this browser is dead by
 * the time this resolves.
 */
export interface ResetPasswordResponse {
  message: string
}

export interface ProfileResponse {
  user: User
}

export interface CasesResponse {
  cases: Case[]
}

/**
 * The body of `POST /cases`, field for field from the `CreateCase` input struct
 * in `backend/handlers/case_handler.go`.
 *
 * Three things about that shape are worth stating, because each one forbids a
 * screen from promising something:
 *
 *  - **There is no category field.** Neither the input nor the `Case` model has
 *    one, so the wizard cannot ask for it and nothing may render it back.
 *  - **`unitId` is optional, and a malformed value is silently dropped.** The
 *    backend parses it as a UUID and only attaches the unit when that succeeds —
 *    an empty or unparseable string still creates the case, just unattached. A
 *    chosen unit is therefore a *request*, not a guarantee, and no copy may say
 *    "dispatched to" a unit on the strength of it.
 *  - **`priority` is passed through untouched.** `isSOS` alone bands P1; the
 *    literal `"high"` bands P2; anything else (including an omitted value) is P3.
 *    So this field is a hint, and `priorityLevel` on the response is the answer.
 */
export interface CreateCaseInput {
  /** A unit id — honoured only if it parses as a UUID. */
  unitId?: string
  title: string
  description: string
  latitude?: number
  longitude?: number
  location?: string
  /** Only the literal `"high"` changes the band. */
  priority?: string
  /** Bands P1 and triggers an immediate dispatch attempt. Only SOS sets it. */
  isSOS?: boolean
  /**
   * File this report anonymously: the precise coordinates and freeform location
   * are zeroed, leaving only a coarse geohash for clustering. Honoured only when
   * the reporter has also enabled location sharing; otherwise the backend treats
   * every report as anonymous (see `POST /cases`).
   */
  hideLocation?: boolean
}

/**
 * The 201 from `POST /cases`.
 *
 * `trackingId` and `priorityLevel` are repeated at the top level *and* on `case`.
 * The receipt reads the top-level pair: they are the values the create actually
 * decided, and reaching into a freshly created object for them invites a screen
 * that disagrees with the server about what it just made.
 */
export interface CreateCaseResponse {
  message: string
  case: Case
  trackingId: string
  priorityLevel: PriorityLevel
}

export interface CaseDetailResponse {
  case: Case
  timeline: CaseTimelineEntry[]
  feedback: CaseFeedback[]
}

export interface ProgressResponse {
  caseId: string
  progress: Progress[]
}

export interface EvidenceListResponse {
  evidence: Evidence[]
}

export interface EvidenceCreateResponse {
  message: string
  evidence: Evidence
}

export interface PresignEvidenceResponse {
  uploadUrl: string
  key: string
  expiresAt: string
}

export interface UnitsResponse {
  units: SecurityUnit[]
}

export interface NearbyUnitsResponse {
  units: UnitWithDistance[]
}

export interface UnitResponse {
  unit: SecurityUnit
}

export interface UnitOfficersResponse {
  officers: UnitOfficer[]
}

export interface CaseAssignmentsResponse {
  assignments: CaseOfficer[]
}

export interface AssignCaseResponse {
  message: string
  assignment: CaseOfficer
  case: Case
}

/** `GET /cases/:id/review` — the case's current state plus its decision history. */
export interface CaseReviewResponse {
  caseId: string
  status: CaseStatus
  reviews: CaseReview[]
}

/** `GET /cases/:id/weekly-updates` — already filtered for the caller's role. */
export interface CaseWeeklyUpdatesResponse {
  caseId: string
  updates: CaseWeeklyUpdate[]
}

/**
 * `POST /cases/:id/weekly-update`. A duplicate week is refused with **409** and a
 * body of `{ error, update }` — i.e. the existing update is returned so the UI can
 * show what is already on file instead of an empty failure.
 */
export interface CaseWeeklyUpdateCreateResponse {
  message: string
  update: CaseWeeklyUpdate
}

export interface CaseWeeklyUpdateConflictResponse {
  error: string
  update: CaseWeeklyUpdate
}

/**
 * `POST /cases/:id/submit-review`. The 409 that a wrong-state submission returns
 * carries the case's actual `status`, so the UI can correct itself instead of
 * guessing why it was refused.
 */
export interface SubmitReviewConflictResponse {
  error: string
  status: CaseStatus
}

export interface NotificationsResponse {
  notifications: Notification[]
  unreadCount: number
}

export interface CaseAnalyticsResponse {
  totalCases: number
  /**
   * NOTE: the backend counts `status = "resolved"`, a state the workflow never
   * sets, so this (and resolutionRate) is always 0 today. Do not present it as
   * truth — prefer deriving open/closed counts client-side.
   */
  resolvedCases: number
  pendingCases: number
  resolutionRate: number
}

/* ------------------------------------------------------------------ */
/* Super-admin surfaces (Wave 10.7a)                                   */
/* ------------------------------------------------------------------ */

/**
 * One row of `GET /admin/users` — a flat map, not a full `User`.
 * Verified against `handlers/super_admin_handler.go:56-70` (GetAllUsers).
 */
export interface SuperAdminUser {
  id: string
  email: string
  phone?: string
  firstName: string
  lastName: string
  role: Role
  status: string
  createdAt?: string
  lastLogin?: string | null
  unitId?: string | null
}

/**
 * Flat envelope from `GET /admin/stats` (`handlers.GetSystemStats` line 351).
 * No nested object — every field is top-level.
 */
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

/** `GET /admin/users` returns `{ users, total }`. */
export interface SuperAdminUsersResponse {
  users: SuperAdminUser[]
  total: number
}

/** One audit-log row from `GET /audit/logs` (`models.AuditLog` JSON tags). */
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
  user?: { id: string; email: string; firstName: string; lastName: string; role: Role }
}

/** `GET /audit/logs` returns `{ auditLogs }`. */
export interface AuditLogsResponse {
  auditLogs: AuditLogEntry[]
}

/** One activity-log row from `GET /audit/activities` (`models.ActivityLog`). */
export interface ActivityLogEntry {
  id: string
  userId: string
  sessionId?: string
  activityType: string
  description?: string
  ipAddress: string
  device?: string
  location?: string
  duration?: number
  createdAt: string
  user?: { id: string; email: string; firstName: string; lastName: string; role: Role }
}

/** `GET /audit/activities` returns `{ activities }`. */
export interface ActivityLogsResponse {
  activities: ActivityLogEntry[]
}

/** System health snapshot from `GET /audit/health` (`models.SystemHealth`). */
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

/** `GET /audit/health` returns `{ health }`. */
export interface SystemHealthResponse {
  health: SystemHealth
}

/** Response body for suspend / activate / role-change mutations. */
export interface UserActionResponse {
  message: string
  user: SuperAdminUser
}
