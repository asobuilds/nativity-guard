# AGENT.md — Nativity Guard

> Operating contract for any AI or human contributor working in this repository.

---

## Who you are

You are an implementation agent for the **Nativity Guard** repository.

You are **NOT** the product owner. You are **NOT** authorized to redesign the architecture. You inspect, plan, implement, test, and report — within the existing architecture.

---

## Source of truth (highest priority first)

1. Existing working code
2. This file (`AGENT.md`)
3. `README.md`
4. Database schema and migrations
5. Existing API contracts
6. Explicit instructions from the user
7. Approved implementation plans

**Rule:** Never assume an apparently reasonable change is architecturally correct. Inspect the repository first.

---

## Security principles

### Core rule

**Unit access is NOT case access.**

An officer in a unit does **not** automatically gain access to every case in that unit. Sensitive case intelligence is available to:
- Assigned officers (`primary` / `paired`)
- Explicitly authorized admins (`CaseAdminAssignment`)
- Head Admin of the unit
- Super Admin

Support officers see a **limited** view of their assigned case only.

### Additional rules

- Sensitive suspect tracking is never a generic officer capability.
- Citizen-facing responses never expose officer identity, evidence internals, admin comments, or other suspects.
- Presumed innocence: a citizen named as a suspect sees only case ID, category, status, and public-safe progress.

---

## Role hierarchy

    Super Admin  (platform-level, outside unit hierarchy)
        ?
        ??? Unit
              ??? Head Admin  (1 per unit, elected by admins)
              ??? Admin       (5–10 per unit, elected by verified members)
              ??? Officer     (tier: primary / paired / support)
              ??? Member      (verified or provisional)

Pre-membership citizens are outside the unit hierarchy.

---

## Case lifecycle (canonical)

    pending ? assigned ? dispatched ? on_scene ? investigating
       ? pending_admin_review
       ? [admin_changes_requested ? investigating]
       ? closed

**Enforced rules:**
- Officers **cannot** close a case directly. Closure requires admin approval.
- Assigned officers **cannot** approve their own case closure.
- Closure requires **2 approvals** from the 2–3 submitted admins.

---

## Governance

### Elections

- Term: 12 months
- Staggered rotation: half the seats renew every 6 months
- Term limit: 2 consecutive terms, then 6-month cooling-off
- Quorum: 50% of eligible verified voters
- Seat bands: 12–20 ? 5 · 21–40 ? 7 · 41–70 ? 9 · 71–100 ? 10
- Vacancy fill: next-highest vote-getter from the last election

### Revocation

- Regular admin removal: **10+ verified votes + Head Admin approval**
- Head Admin removal: **majority of verified unit members**
- One vote per member per cycle
- Votes are immutable; tally is derived, never trusted from the client
- Successful removal triggers cooling-off

### UnitAuth

- Per-unit policy document: thresholds, quorum, seat bands, term rules, cooling-off
- Every election and revocation cycle snapshots the policy version at open time
- No retroactive rule changes

---

## Location model

- Map is a **central platform capability**, not a role-specific feature
- Public map: public-safe geographic information only
- Operational map: restricted by role and case assignment
- Sensitive case location: restricted to authorized case personnel

---

## Living Local Intelligence

    user location ? local context ? short question
        ? community observation ? validation ? knowledge fact
        ? retrieval ? AI reasoning

**Rule:** Nativity Guard learns **facts** before it learns **models**.

Never treat an AI inference as equivalent to a verified fact. All knowledge carries provenance, confidence, source type, and timestamps.

---

## Working rules

1. Inspect before modifying.
2. Never modify unrelated files.
3. Never overwrite architecture without approval.
4. Never invent APIs.
5. Never invent database fields when existing fields can be reused.
6. Never expose secrets.
7. Never commit secrets.
8. Never bypass authorization.
9. Run relevant tests after implementation.
10. Report every modified and created file.
11. Report tests executed and results.
12. Stop after completing the assigned task.
13. Do not automatically begin the next phase.
14. Do not perform destructive Git operations.
15. Do not commit unless explicitly instructed.

---

## Execution cycle

    READ ? UNDERSTAND ? PLAN ? REPORT
        ? WAIT FOR APPROVAL
        ? IMPLEMENT ? TEST ? REPORT ? STOP

---

## Before every task, provide

- OBJECTIVE (one sentence)
- FILES TO INSPECT
- FILES TO MODIFY
- FILES TO CREATE
- FILES NOT TO TOUCH
- IMPLEMENTATION PLAN
- SECURITY CONSIDERATIONS
- TEST PLAN
- RISKS

## After every task, provide

- FILES CHANGED
- FILES CREATED
- WHAT CHANGED
- TESTS RUN
- TEST RESULTS
- REMAINING RISKS
- NEXT POSSIBLE STEP

---

## Testing & Git

- Backend: `go build ./...`, `go vet ./...`, `go test ./...` must all pass
- Frontend: `npm run build` must pass
- No commit without explicit instruction
- No destructive Git ops (no force-push, no rebase, no reset --hard, no drop)

---

## Current production state

- Backend and frontend are deployed on Render; PostgreSQL is hosted on Supabase.
- Production domain: `nativityguard.com`.
- Community reads and writes are live. The feed is scoped server-side to the caller's own place (locality → LGA → state → country → world) and replies are gated by that same visibility rule; the former `VITE_COMMUNITY_WRITES_ENABLED` gate is retired. Never trust a client-supplied place name — the server derives it (decisions D-011…D-016 in `resolution.md`).
- Super-admin navigation must expose only real, usable capabilities. Do not reintroduce fake/Coming Soon fallbacks for implemented admin routes.
- Unit analytics authorization is scoped: super admins may inspect any unit; unit admins may inspect only their own unit.
- Super-admin user mutations already block self-role-change/self-suspension and validate role values.
- Next security work: global settings authorization, last-super-admin protection, RSVP concurrency/capacity safety, real Community moderation, finance-route review, and final role-by-role authorization audit.
- Every production-facing control must be wired to a real backend action or removed/clearly unavailable. Never present mock/demo behavior as live.

---

## Current phase rules

- One Kilo task per file (or two independent files max)
- Build + vet after every task
- Push every 5–10 tasks
- Docs update in the same commit as the change they describe
- Never let README.md or AGENT.md drift from reality

--- END AGENT.md ---

### Navigation regression guard (October 2026)

Preserve the existing carved-mask public background and all five user-selectable themes unless explicitly requested otherwise. In `frontend/src/components/layout/AppShell.tsx`, keep the mobile drawer, its close (X) button, backdrop, menu trigger, bottom navigation, and desktop docked sidebar aligned to the same `lg` breakpoint. Test close by X, backdrop, Escape and navigation, including medium-width tablet layouts and long super-admin menus. Do not describe UI as browser-verified solely because frontend CI passes. PR #63 is the regression repair; validate deployment status before claiming it is live.

### Auth visual regression check

Keep `LoginPage.tsx` and `SignupPage.tsx` desktop brand panels aligned with the shared `ng-public-scene` artwork and five theme palettes. Do not alter signup validation, terms consent or authentication flows for visual-only changes. Test signed-out auth pages separately from the signed-in AppShell drawer; these are different components.

---

## New: Media Attachment Rules (RES-020)

### Refcount & Dedup

- **MediaLink model**: unique index on `(object_key, parent_type, parent_id)`; `OnConflict: DoNothing` makes repeat attach idempotent
- **Content-addressed dedup**: Identical bytes under the same category share one key across different parents. Deleting one parent must **not** remove the key if another parent still references it (MEDIA-ATTACHMENTS-PLAN.md §5.3)
- **Two private storage categories**: `sos_media/` and `report_media/`
- **Size caps**: Voice note ≤ 5 MB (reuses `MaxImageSize`); Video ≤ 20 MB (1/5 of evidence cap)
- **MIME allowlist**: Audio: `audio/mpeg`, `audio/wav`, `audio/ogg`, `audio/webm`; Video: `video/mp4`, `video/webm`, `video/quicktime`

### Presign Endpoints

- `POST /sos/media/presign` — caller-scoped (the alert does not exist yet); gated by `requireMinorApproved`; body `{filename, contentType, sizeBytes, sha256}`; caps: video 20 MB, audio 5 MB
- `POST /community/posts/:id/report/presign` — gated by `communityViewablePost`; same body shape; only a reporter who can already see the post may presign media for it

### Serve Authorization

- `sos_media` → require `?sosId=` + `sosReadScope(db, user)` check (from `handlers/sos_tracking_handler.go:74-87`)
- `report_media` → require `?reportId=` + `postVisibleTo(user, post, nil, nil)` **or** `canModeratePost(user, post, place)`

### SOS Media Attach

- `SendSOSAlert` accepts optional `voiceNoteKey` / `videoKey`; the handler Head-verifies prefix+size+MIME before persisting on the `SOSAlert` row (mirrors `ConfirmEvidenceUpload` at `evidence_presign_handler.go:220-239`)

### Report Media Attach

- `ReportForumPost` accepts optional `voiceNoteKey` / `videoKey`; Head-verifies prefix+size+MIME before creating the `Report` row

### Moderation & Notification (RES-019)

- Existing `canModeratePost` / `isCommunityModerator` predicates unchanged
- `community_report_notification` flow: report filed → moderator notified; only moderators who can act on the post receive notification

### Regression Guards (Must Remain Prominent)

- **Navigation regression**: Preserve carved-mask public background and all five user-selectable themes; mobile drawer at `lg` breakpoint with X, backdrop, Escape, navigation (PR #63 guard). Do not reintroduce fake/Coming Soon routes.
- **Auth visual regression**: `LoginPage.tsx` and `SignupPage.tsx` desktop panels aligned with `ng-public-scene` artwork and five theme palettes. Do not alter signup validation, terms consent or authentication flows for visual-only changes. Test signed-out auth pages separately from the signed-in AppShell drawer.
- **No client-supplied place names**: Server derives place from verified coordinates (locality → LGA → state → country → world). Decisions D-011…D-016 in `resolution.md`.
- **Production state guards**: Every production-facing control must be wired to a real backend action or removed/clearly unavailable.
- **Unit analytics authorization**: Super admins inspect any unit; unit admins inspect only their own.
- **Super-admin user mutation blocks**: Self-role-change/self-suspension blocked; role values validated.
- **No fake/Coming Soon routes**: All admin routes must expose real backend capabilities.

### Landing Page Guest Exploration & Action-Aware Auth (RES-022)

- Public data hooks (`usePublicCases`, `usePublicLeaderboard`, `usePublicBlueprint`, `usePublicSupporters`) provide access to `/public/*` endpoints with instant fallback caches for cold backend resilience.
- `usePendingAction` preserves guest drafts and target destinations across login/signup without storing sensitive credentials.
- `ActionAuthModal` provides contextual action gating when official operations (SOS broadcast, incident report, unit enlistment, voting) are triggered.

---