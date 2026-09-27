/**
 * The mock API route table.
 *
 * These deliberately mirror the *real* backend contract — envelopes, status
 * codes and authorization — so turning `VITE_USE_MOCKS` off changes only where
 * the data comes from, never how the frontend talks to it.
 *
 * Guards the UI depends on, mirrored here:
 *  - progress may only be added while a case is `dispatched`, `on_scene` or
 *    `investigating`;
 *  - evidence may not be attached to a closed case;
 *  - closure is an *approval*: `submit-review` then `approve`, never a `close`;
 *  - an assigned officer cannot approve their own case's closure;
 *  - approve / request-changes both require a non-empty comment;
 *  - a second weekly update for the same UTC week is refused, returning the first;
 *  - `POST /cases` requires a title and a description, bands priority from
 *    `isSOS` / `priority`, attaches a unit only if the id parses, and always
 *    stores the case public;
 *  - `GET /cases` is scoped by role server-side, never by the client.
 */

import { sleep, type MockRoute } from './adapter'
import { communityRoutes } from './community'
import { USERS, seedDatabase, type MockDatabase } from './seed'
import {
  MOCK_ACCOUNTS,
  MOCK_RESET_CODE,
  mockRefreshFor,
  mockTokenFor,
  mockUid,
  userIdFromRefreshToken,
  userIdFromToken,
} from './config'
import { ISO_WEEK_MS, startOfIsoWeekUtc } from '@/lib/week'
import { MIN_SIGNUP_AGE, ageOn } from '@/lib/signup'
import { normaliseResetCode, MIN_RESET_PASSWORD_LENGTH } from '@/lib/passwordReset'
import type {
  Case,
  CaseReview,
  CaseWeeklyUpdate,
  CreateCaseInput,
  PriorityLevel,
  Progress,
  SendSosInput,
  SosAlert,
  User,
} from '@/types/api'

const db: MockDatabase = seedDatabase()
const sosAlerts: SosAlert[] = []
const demoTransactions = [
  { id: 'tx-1', label: 'Community support pledge', amount: 25000, status: 'pending' },
  { id: 'tx-2', label: 'Equipment allocation', amount: 12000, status: 'approved' },
]
const demoFinance = { account: 'Surulere demo operating account', donations: 25000, budget: 100000 }
const demoSettings = { incidentTemplate: 'Record location, incident details and response actions.', retentionDays: 90 }
const demoAudit: { id: string; actor: string; action: string; entity: string; time: string }[] = []

function currentUser(request: Request): User | null {
  const header = request.headers.get('Authorization') ?? ''
  const userId = userIdFromToken(header.replace(/^Bearer\s+/i, ''))
  return userId ? USERS[userId] ?? null : null
}

const unauthorized = { status: 401, body: { error: 'authentication required' } }
const forbidden = (message: string) => ({ status: 403, body: { error: message } })
const notFound = (message: string) => ({ status: 404, body: { error: message } })

/**
 * The unit an officer belongs to, and the unit the administrator runs.
 *
 * The real `/auth/profile` carries no `unitId` yet (a documented contract gap),
 * so the mock stands in for that scoping: an officer sees their own unit's work,
 * and a unit administrator sees the unit they run — Surulere Central, whose
 * contact person is the `admin@shield.ng` account. A super administrator is the
 * only role that sees across units.
 */
const OFFICER_UNIT = seedDatabase().cases[0].unitId
const ADMIN_UNIT = OFFICER_UNIT

function casesVisibleTo(user: User): Case[] {
  switch (user.role) {
    case 'citizen':
      return db.cases.filter((c) => c.reportedBy === user.id)
    case 'officer':
      return db.cases.filter((c) => c.unitId === OFFICER_UNIT || c.assignedTo === user.id)
    case 'unit_admin':
      return db.cases.filter((c) => c.unitId === ADMIN_UNIT)
    case 'super_admin':
    default:
      return db.cases
  }
}

function canSeeCase(user: User, caseItem: Case): boolean {
  return casesVisibleTo(user).some((c) => c.id === caseItem.id)
}

/**
 * Mirrors the backend's `isCaseAdmin`: a super admin, or the administrator of the
 * case's own unit. The real handler resolves this through an active `UnitMembership`
 * row with role `admin`; the mock stands that in with the account's role plus the
 * unit it runs, since `/auth/profile` carries no membership list yet.
 */
function isCaseAdmin(user: User, caseItem: Case): boolean {
  if (user.role === 'super_admin') return true
  return user.role === 'unit_admin' && caseItem.unitId === ADMIN_UNIT
}

/** Mirrors the backend's `isAssignedOfficer`. */
function isAssignedOfficer(user: User, caseItem: Case): boolean {
  return caseItem.assignedTo != null && caseItem.assignedTo === user.id
}

/** Who may read a case's review history and weekly updates. */
function canReviewCase(user: User, caseItem: Case): boolean {
  return (
    isCaseAdmin(user, caseItem) || isAssignedOfficer(user, caseItem) || caseItem.reportedBy === user.id
  )
}

function haversineKm(aLat: number, aLng: number, bLat: number, bLng: number): number {
  const R = 6371
  const toRad = (deg: number) => (deg * Math.PI) / 180
  const dLat = toRad(bLat - aLat)
  const dLng = toRad(bLng - aLng)
  const lat1 = toRad(aLat)
  const lat2 = toRad(bLat)
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLng / 2) ** 2
  return 2 * R * Math.asin(Math.sqrt(h))
}

const nowIso = () => new Date().toISOString()

/**
 * What a `uuid.UUID` field serialises to when the backend never assigned it.
 *
 * `models.Case.UnitID` is `not null` and `CreateCase` leaves it zero when the
 * submitted `unitId` does not parse, so Go marshals the zero value rather than
 * omitting the key. A case filed with no resolvable unit carries this string.
 */
const ZERO_UNIT_ID = '00000000-0000-0000-0000-000000000000'

/**
 * Accounts created through `POST /auth/register` during this session.
 *
 * Kept apart from `MOCK_ACCOUNTS` on purpose: that list is the login screen's
 * one-tap demo buttons, and a registered account landing in it would add a
 * duplicate "Citizen" button. Registration still has to be *usable*, though — the
 * point of the flow is signing in with what you just created — so login consults
 * both lists. Session-only, like the rest of the mock state.
 */
const registeredAccounts = new Map<string, { password: string; userId: string }>()

/**
 * Advances on every rotation. The real service invalidates the refresh token it
 * just consumed; this mock only moves the generation forward, which is enough to
 * catch a client that stores the access token but drops the replacement refresh
 * token — the failure mode rotation exists to expose — but not a double-spend.
 */
let refreshGeneration = 0

/**
 * The account the most recent password-reset request resolved to.
 *
 * The real flow's indirection matters and is modelled: `/auth/reset-password` is
 * handed only a code, and the server finds the user through the persisted reset
 * row — it never sees the identifier again. This stands in for that row. Only the
 * most recent request is kept because the mock recognises a single code; the real
 * service would still honour an earlier, unexpired one.
 */
let pendingReset: { userId: string } | null = null

/**
 * Mirrors `RequestReset`'s lookup: email is case-insensitive, phone matches on
 * digits alone. Unknown identifiers resolve to `null` — the caller must not be
 * able to tell that apart from a match.
 */
function findUserId(identifier: string): string | null {
  const trimmed = identifier.trim()
  if (trimmed.includes('@')) {
    const email = trimmed.toLowerCase()
    const demo = MOCK_ACCOUNTS.find((account) => account.email.toLowerCase() === email)
    if (demo) return demo.userId
    return registeredAccounts.get(email)?.userId ?? null
  }

  const digits = normaliseResetCode(trimmed)
  if (!digits) return null
  // Comparison is on the stored number's digits, like the `phone = ?` clause.
  const match = Object.values(USERS).find(
    (user) => normaliseResetCode(user.phone ?? '') === digits,
  )
  return match?.id ?? null
}

/**
 * Replace a password wherever that account lives.
 *
 * Without this the walk-through would be a lie about the one thing it exists to
 * show: reset the password, then be refused at sign-in for using it.
 */
function setPasswordFor(userId: string, password: string): void {
  const demo = MOCK_ACCOUNTS.find((account) => account.userId === userId)
  if (demo) {
    demo.password = password
    return
  }
  for (const [email, record] of registeredAccounts) {
    if (record.userId === userId) registeredAccounts.set(email, { ...record, password })
  }
}

/** The registration body, field for field from `Register`'s input struct. */
interface RegisterInput {
  email: string
  phone: string
  firstName: string
  lastName: string
  password: string
  dateOfBirth: string
}

export const handlers: MockRoute[] = [
  ...communityRoutes,
  {
    method: 'GET', path: '/demo/admin/state',
    respond({ request }) {
      const user = currentUser(request)
      if (!user) return unauthorized
      if (!['unit_admin', 'super_admin'].includes(user.role)) return forbidden('Administrator access required')
      const unit = db.units[0]
      const cases = user.role === 'super_admin' ? db.cases : db.cases.filter((c) => c.unitId === unit.id)
      const officers = user.role === 'super_admin' ? db.officers : db.officers.filter((o) => o.unitId === unit.id)
      return { body: { cases, officers, units: user.role === 'super_admin' ? db.units : [unit], transactions: demoTransactions, finance: demoFinance, settings: demoSettings, audit: user.role === 'super_admin' ? demoAudit : undefined, demo: true } }
    },
  },
  {
    method: 'PUT', path: '/demo/admin/:section',
    async respond({ request, params }) {
      const user = currentUser(request)
      if (!user) return unauthorized
      if (!['unit_admin', 'super_admin'].includes(user.role)) return forbidden('Administrator access required')
      if (['units', 'settings'].includes(params.section) && user.role !== 'super_admin') return forbidden('Platform administrator access required')
      const input = await request.json() as Record<string, unknown>
      const value = (key: string) => typeof input[key] === 'string' ? String(input[key]).trim() : ''
      const bad = (message: string) => ({ status: 400, body: { error: message } })
      let entity = ''
      if (params.section === 'officers') {
        const name = value('name'), badgeNumber = value('badgeNumber')
        if (!name || !badgeNumber) return bad('Officer name and badge number are required')
        const officer = db.officers.find((o) => o.id === input.id)
        if (officer && officer.unitId !== db.units[0].id && user.role !== 'super_admin') return forbidden('Officer belongs to another unit')
        if (!officer && db.officers.some((o) => o.badgeNumber === badgeNumber)) return bad('Badge number already exists')
        const updated = { ...(officer ?? { id: crypto.randomUUID(), unitId: db.units[0].id, joinedDate: new Date().toISOString() }), name, badgeNumber, rank: value('rank') || 'Officer', role: value('role') || 'patrol', status: value('status') || 'active', phone: value('phone') }
        if (officer) Object.assign(officer, updated)
        else db.officers.push(updated)
        entity = updated.id
      } else if (params.section === 'unit' || params.section === 'units') {
        const unit = params.section === 'unit' ? db.units[0] : db.units.find((u) => u.id === input.id)
        if (input.delete === true) {
          if (!unit || db.cases.some((c) => c.unitId === unit.id) || db.officers.some((o) => o.unitId === unit.id)) return bad('Only empty units can be deleted')
          db.units.splice(db.units.indexOf(unit), 1)
          entity = unit.id
        } else {
          const radius = Number(input.operationalRadius)
          if (!value('name') || !Number.isFinite(radius) || radius <= 0 || radius > 100) return bad('Name and radius between 0 and 100 km are required')
          const latitude = Number(input.latitude), longitude = Number(input.longitude)
          if (!Number.isFinite(latitude) || Math.abs(latitude) > 90 || !Number.isFinite(longitude) || Math.abs(longitude) > 180) return bad('Valid latitude and longitude are required')
          const updated = { ...(unit ?? { ...db.units[0], id: crypto.randomUUID(), isVerified: false, verificationStatus: 'pending' }), name: value('name'), operationalRadius: radius, latitude, longitude, state: value('state'), city: value('city'), contactPhone: value('contactPhone'), contactEmail: value('contactEmail'), status: value('status') || 'active' }
          if (unit) Object.assign(unit, updated)
          else db.units.push(updated)
          entity = updated.id
        }
      } else if (params.section === 'finance') {
        const budget = Number(input.budget)
        if (!Number.isFinite(budget) || budget < 0) return bad('Budget must be a nonnegative number')
        demoFinance.budget = budget
        entity = 'budget'
      } else if (params.section === 'transactions') {
        const tx = demoTransactions.find((t) => t.id === input.id)
        if (!tx || tx.status !== 'pending' || !['approved', 'rejected'].includes(value('status'))) return bad('Select a pending transaction and a decision')
        tx.status = value('status')
        entity = tx.id
      } else if (params.section === 'settings') {
        const retentionDays = Number(input.retentionDays)
        if (!value('incidentTemplate') || !Number.isInteger(retentionDays) || retentionDays < 1 || retentionDays > 3650) return bad('Template and retention between 1 and 3650 days are required')
        Object.assign(demoSettings, { incidentTemplate: value('incidentTemplate'), retentionDays })
        entity = 'settings'
      } else return notFound('Unknown demo section')
      demoAudit.unshift({ id: crypto.randomUUID(), actor: user.email, action: `${params.section} updated`, entity, time: new Date().toISOString() })
      return { body: { ok: true } }
    },
  },
  /* ---------------------------------------------------------------- auth */

  {
    method: 'POST',
    path: '/auth/login',
    async respond({ request }) {
      await sleep(250)
      const body = (await request.json().catch(() => ({}))) as {
        identifier?: string
        email?: string
        password?: string
      }
      // The real handler binds `identifier` and the legacy `email` and falls back
      // from one to the other, so the mock accepts either spelling too.
      const identifier = (body.identifier || body.email || '').trim().toLowerCase()

      const account = MOCK_ACCOUNTS.find((a) => a.email.toLowerCase() === identifier)
      const registered = registeredAccounts.get(identifier)
      const userId = account?.userId ?? registered?.userId
      const expected = account?.password ?? registered?.password

      if (!userId || expected !== body.password) {
        return { status: 401, body: { error: 'invalid credentials' } }
      }

      const user = USERS[userId]
      return {
        body: {
          token: mockTokenFor(user.id),
          // Login always issues a pair, and the backend 500s rather than omit it.
          refreshToken: mockRefreshFor(user.id),
          user: {
            id: user.id,
            email: user.email,
            firstName: user.firstName,
            lastName: user.lastName,
            role: user.role,
          },
        },
      }
    },
  },

  {
    method: 'POST',
    path: '/auth/refresh',
    async respond({ request }) {
      await sleep(200)
      const body = (await request.json().catch(() => ({}))) as { refreshToken?: string }

      const userId = userIdFromRefreshToken(body.refreshToken ?? null)
      const user = userId ? USERS[userId] : null
      if (!user) {
        return { status: 401, body: { error: 'invalid or expired refresh token' } }
      }

      refreshGeneration += 1
      return {
        body: {
          token: mockTokenFor(user.id),
          refreshToken: mockRefreshFor(user.id, refreshGeneration),
        },
      }
    },
  },

  {
    method: 'POST',
    path: '/auth/register',
    async respond({ request }) {
      await sleep(400)
      const body = (await request.json().catch(() => ({}))) as Partial<RegisterInput>

      const email = (body.email ?? '').trim().toLowerCase()
      const phone = (body.phone ?? '').trim()
      const firstName = (body.firstName ?? '').trim()
      const lastName = (body.lastName ?? '').trim()
      const password = body.password ?? ''
      const dateOfBirth = (body.dateOfBirth ?? '').trim()

      if (!email || !phone || !firstName || !lastName || !password || !dateOfBirth) {
        return {
          status: 400,
          body: {
            error:
              'email, phone, firstName, lastName, password and dateOfBirth are required',
          },
        }
      }

      // `validateDOB` runs first on the real handler, then the uniqueness check,
      // then the blank-phone guard. Mirrored in that order so a given request
      // fails for the same reason in either mode.
      const age = ageOn(dateOfBirth)
      if (age === null) return { status: 400, body: { error: 'Invalid date of birth' } }
      if (age < 0) {
        return { status: 400, body: { error: 'Date of birth cannot be in the future' } }
      }
      if (age > 120) {
        return { status: 400, body: { error: 'Date of birth is not valid' } }
      }
      if (age < MIN_SIGNUP_AGE) {
        return {
          status: 400,
          body: { error: 'Registration refused: users under 16 are not permitted' },
        }
      }

      // One account per email or phone, case-insensitive — and, like the real
      // handler, the refusal never says which of the two matched.
      const taken = (candidate: string) =>
        Object.values(USERS).some(
          (u) => u.email.toLowerCase() === candidate || u.phone === candidate,
        )
      if (taken(email) || taken(phone)) {
        return {
          status: 409,
          body: { error: 'An account with that email or phone number already exists' },
        }
      }

      const id = mockUid('aaaa2222', registeredAccounts.size + 1)
      const timestamp = nowIso()
      const user: User = {
        id,
        email,
        phone,
        firstName,
        lastName,
        // Hardcoded on the real handler too, which ignores any `role` sent.
        role: 'citizen',
        status: 'active',
        createdAt: timestamp,
        updatedAt: timestamp,
      }
      USERS[id] = user
      registeredAccounts.set(email, { password, userId: id })

      // 201 with no token: registration deliberately does not sign you in.
      return {
        status: 201,
        body: {
          message: 'User registered successfully',
          user: { id, email, firstName, lastName, role: user.role },
        },
      }
    },
  },

  {
    method: 'POST',
    path: '/auth/forgot-password',
    async respond({ request }) {
      await sleep(350)
      const body = (await request.json().catch(() => ({}))) as { identifier?: string }
      const identifier = (body.identifier ?? '').trim()
      if (!identifier) return { status: 400, body: { error: 'identifier is required' } }

      // An unknown identifier is a silent no-op in `RequestReset`, and the status
      // is 200 either way — so nothing here can be used to discover whether an
      // account exists. An earlier request is deliberately left standing, because
      // the real service does not invalidate a code it already issued.
      const userId = findUserId(identifier)
      if (userId) pendingReset = { userId }

      // The real 200 says "password reset link". Kept verbatim so the mock is a
      // faithful stand-in — but the UI does not display it, because what arrives
      // is a 6-digit code and promising a link sends people looking for one that
      // never comes.
      return {
        body: {
          message:
            "If an account with that email or phone number exists, we've sent a password reset link.",
        },
      }
    },
  },

  {
    method: 'POST',
    path: '/auth/reset-password',
    async respond({ request }) {
      await sleep(400)
      const body = (await request.json().catch(() => ({}))) as {
        token?: string
        newPassword?: string
      }
      const code = normaliseResetCode(body.token ?? '')
      const newPassword = body.newPassword ?? ''

      // `ResetWithToken` checks the length before it looks the token up, so the
      // shorter refusal is the one that wins in either mode.
      if (newPassword.length < MIN_RESET_PASSWORD_LENGTH) {
        return { status: 400, body: { error: 'password must be at least 8 characters' } }
      }

      const reset = pendingReset
      if (code !== MOCK_RESET_CODE || !reset) {
        return { status: 400, body: { error: 'invalid or expired reset token' } }
      }

      setPasswordFor(reset.userId, newPassword)
      // The real service revokes every session and refresh token at this point.
      // Mock tokens carry no revocation list, so that half is not modelled — the
      // reset screen signs itself out on success, which it must do against the
      // real API as well.
      pendingReset = null

      return {
        body: {
          message: 'Password reset successful. You can now log in with your new password.',
        },
      }
    },
  },

  {
    method: 'GET',
    path: '/auth/profile',
    async respond({ request }) {
      await sleep(120)
      const user = currentUser(request)
      if (!user) return unauthorized
      return { body: { user } }
    },
  },
  {
    method: 'PUT', path: '/demo/profile',
    async respond({ request }) {
      const user = currentUser(request)
      if (!user) return unauthorized
      const input = await request.json() as Record<string, unknown>
      const firstName = String(input.firstName ?? '').trim(), lastName = String(input.lastName ?? '').trim()
      const phone = String(input.phone ?? '').trim(), photoUrl = String(input.photoUrl ?? '').trim()
      if (!firstName || !lastName || (phone && !/^\+?[0-9 ()-]{7,20}$/.test(phone))) return { status: 400, body: { error: 'Enter a name and a valid contact number' } }
      if (photoUrl && (!/^https:\/\//.test(photoUrl) || photoUrl.length > 2048)) return { status: 400, body: { error: 'Photo must be an HTTPS image URL' } }
      Object.assign(user, { firstName, lastName, phone, photoUrl, updatedAt: new Date().toISOString() })
      return { body: { user } }
    },
  },

  /* --------------------------------------------------------------- cases */

  {
    method: 'GET',
    path: '/cases',
    async respond({ request }) {
      await sleep(220)
      const user = currentUser(request)
      if (!user) return unauthorized

      const cases = casesVisibleTo(user).map((c) => ({
        ...c,
        evidence: db.evidence.filter((e) => e.caseId === c.id),
        progress: db.progress.filter((p) => p.caseId === c.id),
      }))
      return { body: { cases } }
    },
  },

  {
    method: 'POST',
    path: '/cases',
    async respond({ request }) {
      await sleep(400)
      const user = currentUser(request)
      if (!user) return unauthorized

      const body = (await request.json().catch(() => ({}))) as Partial<CreateCaseInput>

      // Gin binds `title` and `description` as `required` and answers 400 with
      // its own validation string. The wizard disables submit on an empty form,
      // so this is the guard for the request that got past it — a direct call,
      // or a draft restored from storage with a field since cleared.
      if (!body.title?.trim() || !body.description?.trim()) {
        return { status: 400, body: { error: 'title and description are required' } }
      }

      // Mirrors CreateCase's banding: `isSOS` → P1, the literal "high" → P2,
      // anything else → P3. `priority` itself is stored exactly as sent, which
      // is why the wizard omits it rather than sending "normal".
      const priorityLevel: PriorityLevel = body.isSOS
        ? 'P1'
        : body.priority === 'high'
          ? 'P2'
          : 'P3'

      /*
       * `generateTrackingID` is `CS-YYYYMMDD-<n>`, where n is a nanosecond clock
       * reading modulo 10000. The *format* is reproduced rather than approximated:
       * a citizen reads this string aloud to a unit and writes it on a form, so a
       * mock that taught a different shape would be teaching the wrong thing.
       *
       * The `% 10000` is why the real ids collide — it is a 4-digit space minted
       * per call, not a sequence — so this mock inherits that flaw on purpose.
       */
      const trackingId = `CS-${new Date().toISOString().slice(0, 10).replace(/-/g, '')}-${Date.now() % 10000}`

      /*
       * A unit attaches only if the id resolves, which is what `uuid.Parse`
       * decides server-side. An unattached case is not a broken one: the queue is
       * scoped by unit, so it is visible to its reporter and to a super admin and
       * to nobody else — which is what a case no unit has been asked to handle
       * should look like.
       */
      const unit = db.units.find((u) => u.id === body.unitId)
      const now = nowIso()
      const caseItem: Case = {
        id: mockUid('eeee5555', db.cases.length + 1),
        unitId: unit?.id ?? ZERO_UNIT_ID,
        reportedBy: user.id,
        title: body.title.trim(),
        description: body.description.trim(),
        location: body.location,
        latitude: body.latitude ?? 0,
        longitude: body.longitude ?? 0,
        gisLatitude: body.latitude ?? 0,
        gisLongitude: body.longitude ?? 0,
        status: 'pending',
        priority: body.priority,
        priorityLevel,
        trackingId,
        // Hardcoded on the real create path: a report cannot be filed privately,
        // so the UI must never imply the reporter chose to keep it off the map.
        isPublic: true,
        createdAt: now,
        updatedAt: now,
      }
      db.cases.push(caseItem)

      /*
       * The real backend writes no timeline row here — only an async audit log,
       * for which this frontend has no endpoint. This entry exists so the
       * reporter's case log has the event it is about: without it, opening a
       * report just filed shows an empty log, and `pending` becomes a status
       * with nothing on record saying how the case got there. If the backend
       * ever writes a real `case_created` row, delete this and let the server's
       * own entry be the first line.
       */
      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'case_created',
        description: 'Report submitted.',
        status: 'pending',
        createdAt: now,
        user,
      })

      /*
       * Not modelled: the real handler fires `triggerImmediateDispatch` for P1 in
       * a goroutine. Nothing here sends `isSOS` — SOS is its own surface with its
       * own arming flow, not a checkbox on a report form — so the branch would
       * never run. It gets modelled when that surface lands.
       */
      return {
        status: 201,
        body: {
          message: 'Case reported successfully',
          case: caseItem,
          trackingId,
          priorityLevel,
        },
      }
    },
  },

  {
    method: 'GET',
    path: '/cases/:id',
    async respond({ request, params }) {
      await sleep(200)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!canSeeCase(user, caseItem)) {
        return forbidden('you are not authorized to view this case')
      }

      return {
        body: {
          case: {
            ...caseItem,
            evidence: db.evidence.filter((e) => e.caseId === caseItem.id),
            progress: db.progress.filter((p) => p.caseId === caseItem.id),
          },
          timeline: db.timeline
            .filter((t) => t.caseId === caseItem.id)
            .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()),
          feedback: db.feedback.filter((f) => f.caseId === caseItem.id),
        },
      }
    },
  },

  /* ------------------------------------------------------- case workflow */

  {
    method: 'POST',
    path: '/cases/:id/feedback',
    async respond({ request, params }) {
      const user = currentUser(request)
      if (!user) return unauthorized
      const caseItem = db.cases.find((item) => item.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (caseItem.reportedBy !== user.id) return forbidden('only the reporter may leave feedback')
      if (caseItem.status !== 'closed') return { status: 409, body: { error: 'feedback is available when the case is closed' } }
      if (db.feedback.some((item) => item.caseId === params.id && item.userId === user.id)) {
        return { status: 409, body: { error: 'feedback already recorded' } }
      }
      const body = await request.json() as { rating?: number; comment?: string }
      if (!Number.isInteger(body.rating) || (body.rating ?? 0) < 1 || (body.rating ?? 0) > 5) {
        return { status: 400, body: { error: 'rating must be between 1 and 5' } }
      }
      const feedback = {
        id: crypto.randomUUID(), caseId: caseItem.id, userId: user.id,
        rating: body.rating as number, comment: typeof body.comment === 'string' ? body.comment.trim().slice(0, 1000) : '',
        createdAt: new Date().toISOString(),
      }
      db.feedback.push(feedback)
      return { status: 201, body: { feedback } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/dispatch',
    async respond({ request, params }) {
      await sleep(200)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (caseItem.status !== 'assigned') {
        return {
          status: 409,
          body: {
            error: 'case must be assigned before it can be dispatched',
            status: caseItem.status,
          },
        }
      }

      caseItem.status = 'dispatched'
      caseItem.dispatchedAt = nowIso()
      caseItem.updatedAt = caseItem.dispatchedAt
      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'dispatched',
        description: 'Officer dispatched.',
        status: 'dispatched',
        createdAt: caseItem.dispatchedAt,
        user,
      })

      return { body: { message: 'case dispatched successfully', case: caseItem } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/arrive',
    async respond({ request, params }) {
      await sleep(200)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (caseItem.status !== 'dispatched') {
        return {
          status: 409,
          body: {
            error: 'case must be dispatched before arrival can be recorded',
            status: caseItem.status,
          },
        }
      }

      caseItem.status = 'on_scene'
      caseItem.arrivedAt = nowIso()
      caseItem.updatedAt = caseItem.arrivedAt
      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'arrived',
        description: 'Officer arrived on scene.',
        status: 'on_scene',
        createdAt: caseItem.arrivedAt,
        user,
      })

      return { body: { message: 'arrival recorded successfully', case: caseItem } }
    },
  },

  /* -------------------------------------------------- closure review loop */
  /* Closure is an approval, not an action an officer takes alone. There is no
     `POST /cases/:id/close` in the real router, and there is none here. */

  {
    method: 'POST',
    path: '/cases/:id/submit-review',
    async respond({ request, params }) {
      await sleep(250)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!isAssignedOfficer(user, caseItem)) {
        return forbidden('only the assigned officer can submit this case for review')
      }

      // The 409 carries the case's actual status so the client can correct itself.
      if (caseItem.status !== 'investigating' && caseItem.status !== 'admin_changes_requested') {
        return {
          status: 409,
          body: {
            error: 'case cannot be submitted for review from its current status',
            status: caseItem.status,
          },
        }
      }

      const body = (await request.json().catch(() => ({}))) as { finalReport?: string }
      // Accept the report from the request, falling back to whatever is already on
      // the case — the contract does not say which, and both readings must work.
      const finalReport = (body.finalReport ?? caseItem.finalReport ?? '').trim()
      if (!finalReport) {
        return { status: 400, body: { error: 'a final report is required' } }
      }

      caseItem.status = 'pending_admin_review'
      caseItem.finalReport = finalReport
      caseItem.updatedAt = nowIso()
      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'submitted_for_review',
        description: 'Final report submitted for closure approval.',
        status: 'pending_admin_review',
        createdAt: caseItem.updatedAt,
        user,
      })

      return { body: { message: 'case submitted for review successfully', case: caseItem } }
    },
  },

  {
    method: 'GET',
    path: '/cases/:id/review',
    async respond({ request, params }) {
      await sleep(180)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!canReviewCase(user, caseItem)) {
        return forbidden('you are not authorized to view this review')
      }

      const reviews = db.reviews
        .filter((r) => r.caseId === caseItem.id)
        .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())

      return { body: { caseId: caseItem.id, status: caseItem.status, reviews } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/review/request-changes',
    async respond({ request, params }) {
      await sleep(250)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!isCaseAdmin(user, caseItem)) {
        return forbidden('only an administrator can request changes to this case')
      }
      if (caseItem.status !== 'pending_admin_review') {
        return {
          status: 409,
          body: {
            error: 'this case is not awaiting an administrative decision',
            status: caseItem.status,
          },
        }
      }

      const body = (await request.json().catch(() => ({}))) as { comment?: string }
      const comment = (body.comment ?? '').trim()
      if (!comment) {
        return { status: 400, body: { error: 'a comment is required' } }
      }

      const review: CaseReview = {
        id: `rv-${Date.now()}`,
        caseId: caseItem.id,
        adminId: user.id,
        decision: 'request_changes',
        comment,
        createdAt: nowIso(),
      }
      db.reviews.push(review)

      caseItem.status = 'admin_changes_requested'
      caseItem.updatedAt = review.createdAt
      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'changes_requested',
        description: 'Closure not approved — more work requested.',
        status: 'admin_changes_requested',
        createdAt: review.createdAt,
        user,
      })

      return { body: { message: 'changes requested successfully', review, case: caseItem } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/review/approve',
    async respond({ request, params }) {
      await sleep(250)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!isCaseAdmin(user, caseItem)) {
        return forbidden('only an administrator can approve this case')
      }
      // The reviewer must not be the responder.
      if (isAssignedOfficer(user, caseItem)) {
        return forbidden('an assigned officer cannot approve their own case closure')
      }
      if (caseItem.status !== 'pending_admin_review') {
        return {
          status: 409,
          body: {
            error: 'this case is not awaiting an administrative decision',
            status: caseItem.status,
          },
        }
      }

      const body = (await request.json().catch(() => ({}))) as { comment?: string }
      const comment = (body.comment ?? '').trim()
      if (!comment) {
        return { status: 400, body: { error: 'a comment is required' } }
      }

      const review: CaseReview = {
        id: `rv-${Date.now()}`,
        caseId: caseItem.id,
        adminId: user.id,
        decision: 'approve',
        comment,
        createdAt: nowIso(),
      }
      db.reviews.push(review)

      caseItem.status = 'closed'
      caseItem.closedAt = review.createdAt
      caseItem.closedBy = user.id
      caseItem.approvedBy = user.id
      caseItem.updatedAt = review.createdAt
      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'closure_approved',
        description: 'Closure approved.',
        status: 'closed',
        createdAt: review.createdAt,
        user,
      })

      return { body: { message: 'case closure approved successfully', review, case: caseItem } }
    },
  },

  /* ------------------------------------------------------ weekly updates */

  {
    method: 'GET',
    path: '/cases/:id/weekly-updates',
    async respond({ request, params }) {
      await sleep(200)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!canReviewCase(user, caseItem)) {
        return forbidden('you are not authorized to view this case')
      }

      // The privacy boundary the contract draws: a reporter who is neither an
      // administrator nor the assigned officer sees only citizen-visible updates.
      // Two roles legitimately get different feeds for the same case.
      const restricted = !isCaseAdmin(user, caseItem) && !isAssignedOfficer(user, caseItem)
      const updates = db.weeklyUpdates
        .filter((u) => u.caseId === caseItem.id)
        .filter((u) => (restricted ? u.citizenVisible === true : true))
        .sort((a, b) => new Date(a.weekStart).getTime() - new Date(b.weekStart).getTime())

      return { body: { caseId: caseItem.id, updates } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/weekly-update',
    async respond({ request, params }) {
      await sleep(250)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')
      if (!isAssignedOfficer(user, caseItem)) {
        return forbidden('only the assigned officer can file a weekly update')
      }
      if (caseItem.status === 'closed') {
        return {
          status: 409,
          body: { error: 'a closed case does not accept further updates', status: caseItem.status },
        }
      }

      const body = (await request.json().catch(() => ({}))) as Partial<CaseWeeklyUpdate>
      const summary = (body.summary ?? '').trim()
      const investigation = (body.investigation ?? '').trim()
      if (!summary || !investigation) {
        return { status: 400, body: { error: 'summary and investigation are required' } }
      }

      // The week is the server's, not the client's: Monday 00:00 UTC.
      const weekStart = startOfIsoWeekUtc(new Date())
      const existing = db.weeklyUpdates.find(
        (u) =>
          u.caseId === caseItem.id &&
          u.officerId === user.id &&
          new Date(u.weekStart).getTime() === weekStart.getTime(),
      )
      if (existing) {
        // A refusal that returns the update already on file, so the client can show
        // what exists instead of an empty failure.
        return {
          status: 409,
          body: { error: 'you have already filed an update for this week', update: existing },
        }
      }

      const update: CaseWeeklyUpdate = {
        id: `wu-${Date.now()}`,
        caseId: caseItem.id,
        officerId: user.id,
        weekStart: weekStart.toISOString(),
        weekEnd: new Date(weekStart.getTime() + ISO_WEEK_MS - 1).toISOString(),
        summary,
        investigation,
        actionsTaken: body.actionsTaken?.trim() || undefined,
        findings: body.findings?.trim() || undefined,
        evidenceSummary: body.evidenceSummary?.trim() || undefined,
        outstandingActions: body.outstandingActions?.trim() || undefined,
        nextSteps: body.nextSteps?.trim() || undefined,
        citizenVisible: true,
        submittedAt: nowIso(),
        createdAt: nowIso(),
      }
      db.weeklyUpdates.push(update)

      return { status: 201, body: { message: 'weekly update filed successfully', update } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/assign',
    async respond({ request, params }) {
      await sleep(200)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')

      const body = (await request.json().catch(() => ({}))) as {
        officerId?: string
        role?: string
      }
      if (!body.officerId) {
        return { status: 400, body: { error: 'officerId is required' } }
      }

      // Mirrors AssignCase: the id is an *officers* id, and it must belong to
      // the case's own unit, or the assignment is refused outright.
      const officer = db.officers.find((o) => o.id === body.officerId)
      if (!officer) return notFound('officer not found')
      if (officer.unitId !== caseItem.unitId) {
        return { status: 400, body: { error: 'officer does not belong to the case unit' } }
      }

      const role = body.role || 'primary'
      const existing = db.assignments.find(
        (a) => a.caseId === caseItem.id && a.officerId === officer.id,
      )

      let assignment = existing
      if (assignment) {
        assignment.role = role
      } else {
        assignment = {
          id: `as-${Date.now()}`,
          caseId: caseItem.id,
          officerId: officer.id,
          role,
          createdAt: nowIso(),
        }
        db.assignments.push(assignment)
      }

      // Only a primary assignment takes ownership of the case (backend rule).
      if (!caseItem.assignedTo || role === 'primary') {
        caseItem.assignedTo = officer.id
        caseItem.assignedAt = nowIso()
        if (caseItem.status === 'pending') caseItem.status = 'assigned'
        caseItem.updatedAt = caseItem.assignedAt
      }

      db.timeline.unshift({
        id: `tl-${Date.now()}`,
        caseId: caseItem.id,
        userId: user.id,
        action: 'case_assigned',
        description: `Assigned to ${officer.rank} ${officer.name} (${officer.badgeNumber}) as ${role}.`,
        status: caseItem.status,
        createdAt: nowIso(),
        user,
      })

      return {
        body: { message: 'case assigned successfully', assignment, case: caseItem },
      }
    },
  },

  {
    method: 'GET',
    path: '/cases/:id/assignments',
    async respond({ request, params }) {
      await sleep(150)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')

      const assignments = db.assignments
        .filter((a) => a.caseId === caseItem.id)
        .sort((a, b) => new Date(a.createdAt ?? 0).getTime() - new Date(b.createdAt ?? 0).getTime())

      return { body: { assignments } }
    },
  },

  /* ------------------------------------------------------------ progress */

  {
    method: 'GET',
    path: '/cases/:id/progress',
    async respond({ request, params }) {
      await sleep(180)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')

      // Mirrors the real handler: ordered oldest-first.
      const progress = db.progress
        .filter((p) => p.caseId === caseItem.id)
        .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())

      return { body: { caseId: caseItem.id, progress } }
    },
  },

  {
    method: 'POST',
    path: '/cases/:id/progress',
    async respond({ request, params }) {
      await sleep(220)
      const user = currentUser(request)
      if (!user) return unauthorized

      const caseItem = db.cases.find((c) => c.id === params.id)
      if (!caseItem) return notFound('case not found')

      const body = (await request.json().catch(() => ({}))) as {
        action?: string
        description?: string
      }
      if (!body.action) {
        return { status: 400, body: { error: 'action is required' } }
      }

      if (
        caseItem.status !== 'dispatched' &&
        caseItem.status !== 'on_scene' &&
        caseItem.status !== 'investigating'
      ) {
        return {
          status: 409,
          body: {
            error: 'progress cannot be added from the current case status',
            status: caseItem.status,
          },
        }
      }

      const progress: Progress = {
        id: `pg-${Date.now()}`,
        caseId: caseItem.id,
        officerId: user.id,
        action: body.action,
        description: body.description,
        createdAt: nowIso(),
      }
      db.progress.push(progress)

      return { status: 201, body: { message: 'case progress added successfully', progress } }
    },
  },

  /* ------------------------------------------------------------ evidence */

  {
    method: 'GET',
    path: '/evidence/case/:caseId',
    async respond({ request, params }) {
      await sleep(180)
      const user = currentUser(request)
      if (!user) return unauthorized

      const evidence = db.evidence
        .filter((e) => e.caseId === params.caseId)
        .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())

      return { body: { evidence } }
    },
  },

  {
    method: 'POST',
    path: '/evidence/upload',
    async respond({ request }) {
      await sleep(250)
      const user = currentUser(request)
      if (!user) return unauthorized

      const body = (await request.json().catch(() => ({}))) as {
        caseId?: string
        type?: string
        fileUrl?: string
        description?: string
        latitude?: number
        longitude?: number
      }

      if (!body.caseId || !body.type || !body.fileUrl) {
        return { status: 400, body: { error: 'caseId, type and fileUrl are required' } }
      }

      const caseItem = db.cases.find((c) => c.id === body.caseId)
      if (!caseItem) return notFound('case not found')
      if (caseItem.status === 'closed') {
        return { status: 409, body: { error: 'evidence cannot be uploaded to a closed case' } }
      }

      const evidence = {
        id: `ev-${Date.now()}`,
        caseId: caseItem.id,
        uploadedBy: user.id,
        type: body.type,
        fileUrl: body.fileUrl,
        description: body.description,
        latitude: body.latitude,
        longitude: body.longitude,
        isVerified: false,
        uploadedAt: nowIso(),
        createdAt: nowIso(),
      }
      db.evidence.push(evidence)

      return { status: 201, body: { message: 'evidence uploaded successfully', evidence } }
    },
  },

  {
    method: 'PATCH',
    path: '/evidence/:id/verify',
    async respond({ request, params }) {
      await sleep(200)
      const user = currentUser(request)
      if (!user) return unauthorized

      const evidence = db.evidence.find((e) => e.id === params.id)
      if (!evidence) return notFound('evidence not found')

      evidence.isVerified = true
      return { body: { message: 'evidence verified successfully', evidence } }
    },
  },

  /* --------------------------------------------------------------- units */

  {
    method: 'GET',
    path: '/units',
    async respond() {
      await sleep(150)
      return { body: { units: db.units } }
    },
  },

  {
    method: 'GET',
    path: '/units/:id/officers',
    async respond({ request, params }) {
      await sleep(150)
      const user = currentUser(request)
      if (!user) return unauthorized

      // Mirrors handlers.GetOfficersByUnit. The handler was implemented long
      // before it was routed; as of 2026-09-26 `GET /units/:id/officers` is
      // registered in routes/routes.go, so this path now matches the live API.
      const officers = db.officers.filter((o) => o.unitId === params.id)
      return { body: { officers } }
    },
  },

  {
    method: 'GET',
    path: '/units/nearby',
    async respond({ url }) {
      await sleep(150)
      const lat = Number(url.searchParams.get('lat'))
      const lng = Number(url.searchParams.get('lng'))
      const radius = Number(url.searchParams.get('radius') ?? 50)

      const units = db.units
        .map((unit) => {
          const distance = haversineKm(lat, lng, unit.latitude, unit.longitude)
          return {
            ...unit,
            distance: Math.round(distance * 100) / 100,
            isInRange: distance <= radius,
          }
        })
        .sort((a, b) => a.distance - b.distance)

      return { body: { units } }
    },
  },

  /* ------------------------------------------------------- emergency SOS */

  {
    method: 'POST',
    path: '/sos/send',
    async respond({ request }) {
      await sleep(180)
      const user = currentUser(request)
      if (!user) return unauthorized
      if (user.role !== 'citizen') return forbidden('only citizens may send an SOS')
      const input = (await request.json()) as SendSosInput
      if (!Number.isFinite(input.latitude) || !Number.isFinite(input.longitude) ||
          Math.abs(input.latitude) > 90 || Math.abs(input.longitude) > 180 ||
          (input.latitude === 0 && input.longitude === 0)) {
        return { status: 400, body: { error: 'a valid location is required' } }
      }
      const id = crypto.randomUUID()
      const alert: SosAlert = {
        id,
        userId: user.id,
        status: 'pending',
        latitude: input.latitude,
        longitude: input.longitude,
        priority: input.priority === 'critical' ? 'critical' : 'high',
        ...(input.unitId ? { unitId: input.unitId } : {}),
        ...(input.emergencyContacts ? { emergencyContacts: input.emergencyContacts } : {}),
        ...(input.medicalInfo ? { medicalInfo: input.medicalInfo } : {}),
        createdAt: new Date().toISOString(),
      }
      sosAlerts.unshift(alert)
      return { status: 201, body: { alert } }
    },
  },
  {
    method: 'GET',
    path: '/sos/my',
    async respond({ request }) {
      await sleep(120)
      const user = currentUser(request)
      if (!user) return unauthorized
      return { body: { alerts: sosAlerts.filter((a) => a.userId === user.id) } }
    },
  },
  {
    method: 'GET',
    path: '/sos/:id',
    async respond({ request, params }) {
      const user = currentUser(request)
      if (!user) return unauthorized
      const alert = sosAlerts.find((a) => a.id === params.id && a.userId === user.id)
      return alert ? { body: { alert } } : notFound('SOS request not found')
    },
  },

  /* ------------------------------------------------------- notifications */

  {
    method: 'GET',
    path: '/mobile/notifications',
    async respond({ request }) {
      await sleep(150)
      const user = currentUser(request)
      if (!user) return unauthorized

      const notifications = db.notifications
        .filter((n) => n.userId === user.id)
        .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())

      return {
        body: {
          notifications,
          unreadCount: notifications.filter((n) => n.status === 'unread').length,
        },
      }
    },
  },

  {
    method: 'PUT',
    path: '/mobile/notifications/:id/read',
    async respond({ request, params }) {
      await sleep(120)
      const user = currentUser(request)
      if (!user) return unauthorized

      const notification = db.notifications.find((n) => n.id === params.id && n.userId === user.id)
      if (!notification) return notFound('notification not found')

      notification.status = 'read'
      return { body: { message: 'notification marked as read' } }
    },
  },

  {
    method: 'PUT',
    path: '/mobile/notifications/read-all',
    async respond({ request }) {
      await sleep(150)
      const user = currentUser(request)
      if (!user) return unauthorized

      db.notifications.forEach((n) => {
        if (n.userId === user.id) n.status = 'read'
      })
      return { body: { message: 'all notifications marked as read' } }
    },
  },
]
