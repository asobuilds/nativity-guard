# Backend handoff from the frontend — 2026-09-27

This is the consolidated backend work list for the current Nativity Guard frontend. It covers the frontend roadmap (F1–F14, T8–T22) and open review branches #35–#40. It is a request for the backend team, **not** an assertion that every mounted route needs new implementation. The route references below were checked against `backend/routes/routes.go` on `main` at `af9ad1a`; runtime and deployment behavior still need separate verification. No backend code is changed by this PR.

## First: access control and data exposure

| Area | Observed gap | Backend completion condition |
|---|---|---|
| SOS responder access (T13) | `GetSOSAlerts` filters officers/unit admins only when `User.UnitID != nil`; an unassigned officer can receive the unfiltered list. `GetSOSAlertByID` checks ownership for citizens only. `UpdateSOSAlertStatus` checks role but not the alert's unit, and accepts an arbitrary status string. | Scope list and detail to the authorized responder's unit or assignment; deny unassigned staff; check scope on each status change; restrict status transitions to documented values; test cross-unit, missing-unit and citizen requests. Return a deliberate authorized view containing only necessary personal/location fields. **Do this before a live responder console.** |
| Suspect self-view (T11; PR #39) | `GET /suspects/me/cases` puts raw `CaseProgress.Description` into `lastProgress`. PR #39 omits it in the UI, but the response still discloses it. | Remove internal descriptions or replace with an explicitly approved suspect-facing progress summary; test response content and linked-user authorization. |
| Public map and units (F10/T15/T21) | `GET /public/cases` and `/public/units` serialize model objects after filtering; other JSON fields may still reveal reporter/staff/contact details or precise coordinates. | Return reviewed public DTOs with field allowlists and privacy tests for anonymous, coarse and precise-location cases before the landing page uses them. |
| Case details (F4) | The reporter response can include full timeline and feedback records even where the displayed UI hides names. | Audit `GET /cases/:id` and associated timeline/feedback JSON; curate fields per role server-side. |
| Case status update (F7/F8) | `PUT /cases/:id` bypasses the review transition checks described in `frontend/frontReadme.md` §A7. | Enforce actor, unit and allowed-transition rules in the backend; reject invalid status jumps and test each role. |
| Unit selection on report (F3) | An omitted or malformed `unitId` can become a zero-UUID case that no unit can triage. The UI requires a unit but the API does not enforce it. | Reject invalid/missing unit IDs or explicitly place unattached reports in a staffed triage queue; test both cases. |

## Missing or insufficient contracts for planned screens

| Frontend area | Existing route / missing capability | Backend deliverable |
|---|---|---|
| T10 elections | Open/vote/close/results routes exist (`/units/:id/elections`, `/elections/:id/*`); governance audit exposes only a privileged recent-ID summary. There is no member-facing election/candidate discovery route. | Provide a role-scoped list of active/eligible elections and each ballot's candidates, eligibility, open/close times and the caller's voting state. Specify vote body and idempotency/error behavior; test nonmembers and duplicate votes. Voting UI remains deferred. |
| T10 revocations | Open/vote/close and `GET /revocations/:id` exist; the overview is aggregate only. | Provide discoverable authorized cycles, affected role/member and vote choices with privacy/eligibility checks, plus permitted actions and decision status. Agree transitions and tests before building revocation controls. |
| T12 invites (PR #40) | `POST /invites` and public `POST /invites/validate` exist. Signup always creates a citizen and does not redeem the code; no invite redemption route is mounted. | Define an atomic code redemption and unit admission/application flow, including signup versus existing user, verification, expiry/use count, replay prevention and role/unit authorization. Expose the post-signup state to the caller. Until then the frontend shows validation and citizen signup only. |
| T22 community moderation (F6) | Posts/replies/announcements/events/RSVP are mounted; the mock report path `POST /community/posts/:id/report` and review queue/actions are absent. | Provide guarded report submission, reviewer queue, decisions and audit log with rate limits, deduplication and access tests. Keep the current demo report button out of a live workflow until this exists. |
| F7 case lifecycle | No mounted transition for `on_scene → investigating`; deescalation review decision has no route. `GetCaseAccountability` exists but is unrouted. | Define authorized transition and deescalation handlers; expose per-case accountability only to allowed roles with curated response. Test lifecycle and visibility. |

## Existing routes needing live integration or contract confirmation

These entries request an end-to-end contract check and fixes if it fails. A mounted route alone does not prove a working deployment. Provide sanitized fixtures/test accounts, roles and a stable test environment; do not use a real emergency for SOS verification.

| Feature(s) | Backend verification requested |
|---|---|
| F1/T8/T9 auth and account | Verify citizen registration/login, password reset delivery and reset session invalidation, `GET/DELETE /auth/sessions`, OTP, unit application/membership and account lifecycle. Correct the forgot-password success copy that says “link” when a six-digit code is sent. Reconcile signup minimum password 6 versus reset minimum 8; update stale comments claiming delivery is unwired. |
| F2/T13 SOS | Verify `/sos/send` returns `{message,sos,escalationTime}` and `/sos/my` yields user-owned history; check notification/escalation, `pending`/`dispatched`/`resolved` lifecycle, medical-info and emergency-contact handling, failed/duplicate sends and staff status updates. The frontend mock currently needs contract reconciliation; a receipt is submission, not dispatch. |
| F3 reporting and evidence | Verify case creation/assignment with valid units, presign → storage PUT → confirm and direct binary upload, evidence authorization, expiry and deletion on the deployed object store. Agree offline submission/idempotency before promising an offline queue. |
| F4/T14 feedback and tracking | `POST /cases/:id/feedback` is mounted; verify reporter-only eligibility, closed-case rule, duplicate handling and response shape. Verify weekly update visibility and reliable notification of case status changes. |
| F5 awareness | Verify alerts/news/subscriptions and `/mobile/notifications*` end to end, including actual push delivery, device registration and consent/permission states. Demo subscriptions are not proof of delivery. |
| F6 community | Verify existing posts/replies/announcements/events/RSVP responses, scope and permissions against live Go; moderation is a separate missing contract above. |
| F7/F8 officer and unit admin | Verify dispatch/arrive, review decisions, progress, weekly narratives, evidence verification, unit roster `GET /units/:id/officers`, named assignment and role/tenant scoping. Demo overview/analytics/finance/settings screens require real data contracts before they are presented as live. |
| F9/T15 super admin | Confirm live user governance, audit, analytics/health, finance/bank/account data, exports and role checks for the existing route families; define missing data/actions per screen with frontend before declaring demo surfaces live. |
| F10 maps | Verify nearby units, aggregated activity, geocoding and access to precise versus coarse coordinates by role; add safe public DTOs as above. |
| F11 communication / F12 AI | Router advertises communication and AI families, but the frontend has no corresponding production screens. Before implementing them, provide authenticated message/call/presence/history and assistant/tip/warning contracts, moderation/retention rules and labeled AI output expectations. Scope these with the frontend team; they are **not** all confirmed missing routes. |
| F13 settings / F14 offline-PWA | Verify live profile/avatar writes, consent, preference and account export/deletion behavior. Agree offline data retention, replay/idempotency and conflict rules before queued SOS or case writes; a service worker alone cannot guarantee emergency delivery. |
| T16–T21 additional domains | Appeals and unit transfers have mounted handlers and frontend PRs; verify cross-unit permissions, decisions and history. Suspect operations/counter-statements/expungement, video/social monitoring, peacebuilding, public donation/finance/leaderboards and ratings each need response-field and actor audits with live fixtures before their frontend screens are called integrated. Video automated findings must only be described as real when a real data source exists. |

## Review order and acceptance

1. Fix the access-control and disclosure items, especially SOS and suspect response exposure. Review the public DTOs before anonymous map previews.
2. Supply election/candidate discovery, revocation discovery, invite redemption, moderation, and case-transition contracts as separate reviewable backend PRs. Include request/response examples and permission tests.
3. Provide a live test environment and run the integration matrix above with the frontend; record what passed. Keep T8 and T13 incomplete until the corresponding live checks pass.

The frontend PRs #35–#40 are review branches, not proof of deployment or live verification. PR #38 explicitly documents deferred T10 voting/revocations; PR #39 omits sensitive suspect progress text from rendering; PR #40 validates invites without claiming automatic unit membership. This handoff asks for backend follow-up and does not authorize backend edits in the frontend branches.
