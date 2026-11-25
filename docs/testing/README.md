# Nativity Guard — Release Testing & Verification

This folder is the project's release-test checkpoint. Use it when a feature set is close to completion, before a production release, and after significant security or database changes.

**Current status:** Planning checklist. Items are not considered passed until a tester records the environment, date, build/commit, evidence, and result.

## How to use this checklist

1. Test against a dedicated staging environment, never live incidents or real emergency operations.
2. Record the exact Git commit, backend/frontend versions, browser/device, database version, and configuration used.
3. Use synthetic accounts and fictional reports, citizens, units, and incidents.
4. Mark each item `[x]` only after the test has actually passed. Use `[blocked]` or leave it unchecked when prerequisites are missing.
5. File a GitHub issue for every failure, include reproducible steps and sanitized logs, and link the issue here.
6. Re-run affected tests after fixes. A passing CI workflow does not replace browser, database, security, or operational testing.

## Test record

Copy this block into a release record for each test cycle:

- Date/time and timezone:
- Tester:
- Environment (local/staging):
- Main commit SHA:
- Backend commit/version:
- Frontend commit/version:
- Database engine/version:
- Browser and version:
- Device/OS:
- Push notification configuration verified (yes/no):
- Test data set:
- Result: NOT STARTED / PASS / FAIL / BLOCKED
- Related issues / evidence links:

---

## 1. Build, CI and deployment readiness

- [x] `backend-ci` passes on the exact release commit (13ee2edde0f88028a02d5eefa55f1cfd77de6535; run 43887298427).
- [x] `frontend-ci` passes on the exact release commit (13ee2edde0f88028a02d5eefa55f1cfd77de6535; run 43887298196).
- [x] `integration-race` passes on the exact release commit (13ee2edde0f88028a02d5eefa55f1cfd77de6535; run 43887298361).
- [ ] All other required repository checks pass; no required check is missing or skipped unexpectedly.
- [ ] Backend starts cleanly with production-like configuration.
- [ ] Frontend builds and loads without console errors.
- [ ] Health/readiness endpoints report the expected dependencies.
- [ ] Configuration and secrets are supplied through the deployment secret manager, not committed files.
- [ ] Rollback procedure and previous known-good release are documented.

## 2. Authentication, accounts and roles

- [ ] Citizen registration, login, logout and session expiry work.
- [ ] Password reset/change and token/session revocation work.
- [ ] Banned and suspended accounts cannot log in or continue using an existing session.
- [ ] Unverified users cannot access verified-member or security-only functions.
- [ ] Unit membership and officer status are derived and checked server-side.
- [ ] Super-admin, unit-admin, head-admin, officer, observer and citizen permissions are tested independently.
- [ ] Direct API requests cannot bypass frontend role restrictions.

## 3. Geographic community chat

- [ ] A citizen is offered only the locality, LGA and state rooms allowed by their verified server-side location.
- [ ] Room membership cannot be widened by changing client-supplied locality, LGA, state or room identifiers.
- [ ] Users in another locality/LGA/state cannot read or post in a restricted room.
- [ ] Messages persist before being delivered and remain available after refresh/re-login.
- [ ] History pagination/cursors return stable pages without duplicates or omissions.
- [ ] Empty, oversized, malformed and unsupported message payloads are rejected safely.
- [ ] Rate limits prevent spam without blocking normal conversation.
- [ ] Polling refreshes new messages and handles offline/reconnect conditions without duplicating messages.
- [ ] Chat layout is usable on narrow mobile screens and desktop.
- [ ] Keyboard navigation, focus visibility, labels and screen-reader semantics are checked.

## 4. Citizen-to-security private conversations

- [ ] A citizen can start a conversation only with an eligible verified active security unit.
- [ ] Only the citizen and explicitly assigned active unit participants can read or send messages.
- [ ] A different citizen cannot discover or read the conversation by guessing its ID.
- [ ] A unit member, unit admin or super-admin does not gain implicit access merely from their role.
- [ ] Removing/deactivating a participant immediately revokes both read and send access.
- [ ] Revoked access is enforced on existing sessions and every relevant API request, not just the UI.
- [ ] Rejoining or reassignment follows an explicit, audited authorization path.
- [ ] Private message content never appears in public feeds, unrelated rooms, logs or notification previews.
- [ ] Attachments remain disabled unless a separately reviewed secure attachment workflow is implemented.

## 5. Security-team operational messaging

- [ ] Only verified active members of the relevant unit can enter its operational room.
- [ ] Joint task-force rooms require current task-force membership and correct assignment.
- [ ] Expired, suspended or removed unit/task-force membership revokes access immediately.
- [ ] Citizens cannot enumerate, subscribe to, or read security operational rooms.
- [ ] Client-supplied room IDs cannot grant access to another unit or task force.
- [ ] Existing push-to-talk/audio and emergency communication workflows still work.
- [ ] Operational messages and privileged actions have the required audit trail.
- [ ] Any super-admin emergency access is explicit, narrowly scoped and audited.

## 6. Reporting, blocking and moderation

- [ ] Citizens can report an inappropriate message and receive a safe confirmation.
- [ ] A user can block another user; blocking behaviour is consistent across supported chat surfaces.
- [ ] Block/report controls cannot be used to access another user's private information.
- [ ] Moderators see only reports within their authorized geographic jurisdiction.
- [ ] Moderators can hide a message or dismiss a report only when authorized.
- [ ] Moderator actions record actor, target, reason, timestamp and scope.
- [ ] Hidden content is handled consistently in history, polling and notifications.
- [ ] Abuse controls, rate limits and anti-spam rules are tested under repeated requests.

## 7. Notifications and delivery

- [ ] In-app notifications are persisted for authorized recipients.
- [ ] Notification links navigate to the correct room/conversation after authentication.
- [ ] A recipient who has lost room access cannot regain access by opening an old notification.
- [ ] Polling fallback delivers updates reliably and handles retries/network interruptions.
- [ ] Web Push is tested in supported browsers with valid VAPID configuration.
- [ ] Permission denied, expired subscription and delivery failure states are handled.
- [ ] Notification previews do not disclose private message content on shared/locked screens unless explicitly approved.
- [ ] Duplicate notifications are not generated by retries or repeated polling.

## 8. Database, migrations and retention

- [ ] All migrations apply cleanly to an empty staging PostgreSQL database.
- [ ] Migrations apply cleanly from the previous supported schema version.
- [ ] New tables, foreign keys, uniqueness constraints and indexes match the intended access patterns.
- [ ] Migration rollback/forward-recovery plan is documented and exercised where supported.
- [ ] Concurrent message writes and membership changes do not create duplicate or inconsistent records.
- [ ] Retention, deletion, moderation and audit-log policies are reviewed and approved.
- [ ] Backups and restore procedures are tested.
- [ ] No production data is copied into test fixtures or logs.

## 9. Existing critical workflows — regression pass

- [ ] SOS creation and authorized responder notification work.
- [ ] SOS voice/video upload, validation, retrieval authorization and deletion behave correctly.
- [ ] Citizen incident/report creation, geographic routing and evidence access work.
- [ ] Unit assignment, dispatch, arrival/geofence and case lifecycle transitions work.
- [ ] Agency observer visibility remains limited to its authorized LGA and category.
- [ ] Officer panic/distress workflow and notifications work.
- [ ] Suspect visibility, archive and approved privacy-expungement workflows remain restricted.
- [ ] Community feed geographic scope, replies, media and moderation still work.
- [ ] Existing LGA convergence/bulletin channels and push-to-talk are not regressed.

## 10. Privacy, security and resilience

- [ ] Cross-citizen, cross-LGA, cross-state and citizen-to-security access-denial tests pass.
- [ ] Authorization is checked on every read, write, subscription and delivery path.
- [ ] SQL injection, stored/reflected XSS, CSRF where applicable, IDOR and mass-assignment tests are performed.
- [ ] User-authored text is rendered as text; untrusted content is never executed as markup.
- [ ] Sensitive tokens, private messages, precise location and personal data are absent from logs and analytics.
- [ ] Rate limiting, payload limits and abuse controls withstand burst/replay tests.
- [ ] Service restart, database interruption and poor-network recovery do not corrupt messages or incident records.
- [ ] Error responses do not expose stack traces, secrets or private records.
- [ ] Security review findings are tracked and resolved or formally accepted before release.

## 11. Human acceptance and release sign-off

- [ ] Citizen representative completes the key reporting and community-chat journeys.
- [ ] Security-unit representative completes private inbox and operational-room journeys.
- [ ] Moderator/admin representative completes report review and moderation journeys.
- [ ] Desktop, Android and iOS browser/device coverage is recorded.
- [ ] Accessibility and responsive checks are completed.
- [ ] Known limitations and operational support/escalation contacts are documented.
- [ ] No open critical/high severity defects remain without explicit risk acceptance.
- [ ] Product owner and technical/security reviewer approve the release.


## Phase 1 — Agency and field-officer operations

### Agency division administration and team management

- [ ] Agency division admin lands on the agency oversight portal; agency observers cannot see or use team-management controls.
- [ ] Division admin can list only observer accounts attached to their server-resolved agency tenant.
- [ ] Division admin can create an observer with valid name, email, phone, rank and temporary password.
- [ ] Missing names, invalid email, missing phone and passwords shorter than six characters are rejected before submission.
- [ ] Duplicate email/phone conflicts are shown clearly; retry does not create duplicate accounts.
- [ ] A newly created observer appears after the roster refresh and can authenticate with the issued credentials.
- [ ] A division admin cannot create or list observers in another tenant by changing request data or identifiers.
- [ ] An observer cannot call observer-management endpoints, even when bypassing the frontend.
- [ ] Team roster errors, loading, empty state and successful creation are usable on desktop and mobile.
- [ ] Temporary credentials are not written to logs, browser storage, analytics or the release test evidence.


### Agency-to-operational-unit linking and scoped case oversight

- [ ] Super admin can link an agency division only to an active, verified operational unit.
- [ ] Frontend eligible-unit selector filters by active status, verification, agency type, exact state/LGA and already-linked unit IDs.
- [ ] Frontend unit-selector tests cover matching, mismatched agency/jurisdiction, inactive/unverified, already-linked and no-tenant states.
- [ ] Linking rejects an agency-type mismatch (police, NSCDC, fire or FRSC).
- [ ] Linking rejects state or LGA mismatch; matching is case-insensitive but jurisdiction values must otherwise agree.
- [ ] One operational unit cannot be linked to more than one agency division.
- [ ] Non-super-admin users cannot create, replace or inspect tenant-to-unit links through the super-admin endpoint.
- [ ] Agency division admin can open cases only for the single operational unit linked to their tenant.
- [ ] Agency case queue returns only cases belonging to the linked unit, never an unfiltered national or LGA-wide feed.
- [ ] Unlinked agency tenant receives no case visibility; cross-unit case IDs are denied by server-side case access middleware.
- [ ] Agency division admin cannot dispatch, arrive, submit officer progress or close a case as an officer.
- [ ] Verify tenant-to-unit unique index and nullable UnitID migration on staging PostgreSQL; test rollback/backup procedure before production.
- [ ] Confirm existing unit head-admin, unit-admin, officer and citizen case permissions remain unchanged.

### Agency operational visibility

- [ ] Agency incident feed is limited to the tenant's exact state and LGA.
- [ ] Agency-type category filtering is enforced server-side (police, NSCDC, fire and FRSC).
- [ ] Out-of-jurisdiction incidents and units never appear in the response.
- [ ] Observer descriptions redact contact details, precise coordinates, URLs and suspected personal names.
- [ ] Active SOS, open-case and responding-unit totals match the synthetic staging records.
- [ ] Pagination, empty results, refresh and API failure states behave correctly.

### Field-officer and dispatch workflow

- [ ] Unit administrators can see only the roster of units they administer.
- [ ] Officer roster creation, update, deactivation and deletion enforce unit-admin authorization.
- [ ] An administrator can assign a primary officer and supporting officers only from the case's own unit.
- [ ] An officer cannot view or act on a case assigned to another officer without an explicit assignment.
- [ ] Assigned officer can dispatch a case and the status/timestamp are persisted.
- [ ] Arrival is accepted only within the configured 200 m geofence when case coordinates exist.
- [ ] An officer cannot dispatch, arrive, or submit progress for a case outside their assignment.
- [ ] Case progress, evidence, weekly updates and final-report submission remain attached to the correct case.
- [ ] Administrator review/approval is required before closure where the workflow requires it.
- [ ] Reassignment, removed officer access, duplicate assignment requests and concurrent updates are tested.
- [ ] Officer queue and case workspace work on narrow mobile screens and with keyboard navigation.

### Agency-to-unit operational integration — required follow-on gate

- [ ] Agency tenant-to-operational-unit relationship is explicitly modelled and migration-tested before agency admins can assign field officers.
- [ ] Agency admin case visibility is derived from the tenant's authorized jurisdiction and agency category on the server.
- [ ] Agency admin cannot assign officers outside the linked unit or tenant jurisdiction.
- [ ] Agency assignment, dispatch and escalation actions are audited with actor, target, scope and timestamp.
- [ ] Cross-agency, cross-LGA and cross-state assignment attempts are denied.
- [ ] Agency access does not grant access to private citizen/security conversations or unrelated unit operational rooms.

## Release decision

**Do not mark the release ready solely because CI is green.** Release readiness requires passing automated checks, staging integration tests, authorization/security tests, database verification, and human acceptance.

- Final decision: NOT READY / READY WITH ACCEPTED RISKS / READY
- Approved commit SHA:
- Outstanding blockers:
- Accepted risks and approver:
- Product owner sign-off:
- Technical sign-off:
- Security sign-off:
- Date:


### Authenticated officer roster linking

- [ ] Unit admin can create an officer roster record only for an existing active user with an active officer membership in that exact unit.
- [ ] The officer roster record ID matches the authenticated officer user ID so dispatch, arrival, progress and panic authorization resolve the same identity.
- [ ] A user with no officer membership, a pending/revoked membership, or a different-unit membership cannot be linked.
- [ ] Duplicate officer account links and duplicate badge numbers are rejected without creating a second roster record.
- [ ] Newly linked officers can be assigned to a case in their unit and dispatch it using their own authenticated session.
- [ ] A citizen, observer, agency admin or unrelated unit member cannot create officer roster records.


---

## Test execution record — 2026-11-16

- Environment: GitHub Actions CI on the pushed `main` commit.
- Main commit SHA: `13ee2edde0f88028a02d5eefa55f1cfd77de6535`.
- Backend/frontend versions: exact versions defined by the workflow files at this commit.
- Result: **AUTOMATED CI PASS; MANUAL STAGING ACCEPTANCE BLOCKED / NOT RUN**.
- No automated check failures were reported for this commit.

### Automated results

| Workflow | Run | Result | Verified steps |
|---|---:|---|---|
| backend-ci | [43887298427](https://github.com/asobuilds/nativity-guard/actions/runs/43887298427) | PASS | Build, vet, unit tests |
| frontend-ci | [43887298196](https://github.com/asobuilds/nativity-guard/actions/runs/43887298196) | PASS | Install dependencies, type check, lint, tests, build |
| integration-race | [43887298361](https://github.com/asobuilds/nativity-guard/actions/runs/43887298361) | PASS | Integration suite with race detector, services integration suite with race detector, concurrency test x3, fuzz smoke (30s per target) |

### Manual officer-account linking and dispatch acceptance

**Status: BLOCKED — NOT EXECUTED.** A staging API/frontend URL, isolated test database, and synthetic admin/officer accounts were not available to this test run. These are not recorded as failures because the tests were not attempted. Execute them in staging and attach sanitized request/response evidence before marking any as passed.

- [blocked] Unit administrator links an existing active officer account with active officer membership in the exact unit.
- [blocked] A user with no officer membership is rejected.
- [blocked] A user whose membership is pending is rejected.
- [blocked] A user whose membership is revoked is rejected.
- [blocked] A user who belongs only to a different unit is rejected.
- [blocked] A user whose account role is not officer is rejected.
- [blocked] Duplicate roster/account linking is rejected and does not create a second roster entry.
- [blocked] The created roster record ID matches the authenticated officer account ID.
- [blocked] The linked officer can sign in, receive an assignment in their own unit, and dispatch using their own authenticated session.
- [blocked] An unrelated officer, citizen, observer, agency admin, or other unauthorized user cannot create roster links or dispatch the case.
- [blocked] The audit record captures the authorized actor and roster-linking action.

### Other release gates not exercised by this CI run

**Status: BLOCKED / NOT RUN — staging, real browser/device, database administration, push configuration, and human test participants are required.**

- [blocked] Agency observer team-management journeys and tenant-isolation checks in a browser against staging.
- [blocked] Agency-to-unit linking, jurisdiction mismatch, duplicate link, and scoped case visibility checks against staging PostgreSQL.
- [blocked] Agency admin assignment/dispatch denial and cross-agency/cross-LGA/cross-state authorization checks.
- [blocked] Citizen locality/LGA/state chat room membership, history pagination, refresh, offline/reconnect, and mobile layout.
- [blocked] Citizen-to-security private conversation membership, guessed-ID access denial, participant revocation, and private-content leakage checks.
- [blocked] Security operational room and task-force membership revocation and authorization checks.
- [blocked] Report, block, moderation jurisdiction, moderator audit trail, and abuse/rate-limit acceptance.
- [blocked] In-app notification navigation, polling fallback, browser Web Push with configured VAPID keys, and expired/denied subscription handling.
- [blocked] Empty-database and previous-schema PostgreSQL migration runs, rollback/forward recovery, backup and restore.
- [blocked] Browser/device acceptance on desktop, Android and iOS, keyboard navigation, screen-reader semantics, and accessibility review.
- [blocked] Production-like configuration, secrets management, deployment health/readiness, rollback rehearsal, and human sign-off.

### Failure log

- Automated failures on commit `13ee2edde0f88028a02d5eefa55f1cfd77de6535`: **none**.
- Manual failures: **none recorded; manual tests above were not run**.
- Follow-up: provision an isolated staging environment and synthetic test accounts, execute the blocked cases, and record actual failures with reproduction steps and sanitized evidence here. Do not change a blocked item to passed based on CI alone.


---

## Follow-up test execution — 2026-11-16 (site availability and latest main CI)

- Main commit under test before this record: `7f63b18df366f97ecb03499d13a1ba15e4659b2c`.
- Reported issue: user reports that the deployed Nativity Guard website cannot be reached.
- Frontend URL checked: https://nativityguard-frontend.onrender.com/.
- External page probe: URL did not load through the available page fetcher. This does not identify the root cause or prove that every user's browser receives the same failure.
- Render platform status page reported all systems operational and no incident for 2026-11-16. Platform-wide outage is therefore not currently supported by the available evidence.
- Deployment result: **BLOCKED — site availability smoke test cannot be completed; investigate the specific Render static-site deployment, custom service events, and deployment logs.** Do not mark the whole application healthy based on CI.

### Latest main CI results (commit `7f63b18df366f97ecb03499d13a1ba15e4659b2c`)

| Workflow | Run | Result at review | Notes |
|---|---:|---|---|
| backend-ci | [43889845969](https://github.com/asobuilds/nativity-guard/actions/runs/43889845969) | PASS | Backend job completed successfully. |
| frontend-ci | [43889846025](https://github.com/asobuilds/nativity-guard/actions/runs/43889846025) | PASS | Frontend job completed successfully. |
| integration-race | [43889845885](https://github.com/asobuilds/nativity-guard/actions/runs/43889845885) | IN PROGRESS | Race integration job had not completed at the time this record was written; do not count it as passed. |

### Failed or blocked acceptance checks

- [blocked] Open the deployed frontend at https://nativityguard-frontend.onrender.com/ and verify that the landing/login page loads. User reports “site cannot be reached”; independent page probe also could not retrieve it. Root cause is undetermined; check the Render service's current status, latest deployment, domain assignment, build output, and static-site logs.
- [blocked] Verify frontend-to-backend connectivity against the deployed backend, including the backend `/health` endpoint and browser network/console errors. Do not expose secrets or tokens in evidence.
- [blocked] Complete the previously listed staging acceptance tests for agency administration, officer account linking/dispatch, geographic chat, private citizen-security messaging, moderation, notifications, migrations/backups, security, and mobile/accessibility. The deployment URL being unavailable prevents end-to-end execution; these are blocked, not passed.

### Historical CI failures that were repaired

- [x] PR #177's earlier integration-race compile failure on `a0aa7149c585d718df5e9d05c4f0b7d3b003d99e` (`handlers/agency_tenant_test.go:636:5: no new variables on left side of :=`) was repaired directly on the PR branch in commit `c29ae86547dc9b19899cd8686f68ec3938a3942d` by changing the duplicate `w :=` declaration to `w =`. The original failed run [43882206377](https://github.com/asobuilds/nativity-guard/actions/runs/43882206377) skipped the remaining integration-race stages. **Verification of the repaired PR head is pending**; do not merge until backend, frontend and integration-race checks pass on the repaired head.
- [x] PR #178 integration-race failed on the earlier head `a67fc0f8166638e0e0849b0b825461929517c85a`; the issue was the duplicate response-recorder declaration in `backend/handlers/agency_tenant_test.go`. It was corrected in `cd3edc01a1823649ec467554d4660b9496771f19`, after which the branch integration-race run passed. See [initial failed run 43884034357](https://github.com/asobuilds/nativity-guard/actions/runs/43884034357) and [repaired run 43885174812](https://github.com/asobuilds/nativity-guard/actions/runs/43885174812).

- Follow-up required: once the Render deployment is reachable, rerun the blocked smoke and end-to-end checks, capture the exact error and sanitized logs, and update this section with PASS/FAIL and evidence. A blocked test is not a failure until it is actually executed and fails.

---

## Topology & Wave-1 Update — 2026-11-25

- **Production topology updated**: The retired Render static-site frontend (`nativityguard-frontend.onrender.com`) was removed in PR #180. Frontend is now hosted on **Vercel** (`https://nativityguard.com`, `https://www.nativityguard.com`), backend API on **Render** (`https://nativityguard-backend.onrender.com` with `ALLOWED_ORIGINS` configured for the Vercel origins), and PostgreSQL on **Supabase**.
- **Wave-1 fixes merged on `main` (PRs #179–#184)**:
  - PR #179: `integration-race.yml` Go version synced to `backend/go.mod`.
  - PR #180: Vercel frontend + Render backend CORS and env configuration (`render.yaml`, `vercel.json`, `frontend/vercel.json`, `frontend/.env.example`).
  - PR #181: Last remaining active `super_admin` protected from deletion, suspension, ban, or demotion.
  - PR #182: Finance route authorization, atomic approvals, and bank account encryption guards hardened.
  - PR #183: `UnitDetailPage` `cases`, `sos`, and `compliance` tabs wired to live views; `ComingSoonPage.tsx` removed.
  - PR #184: Unreferenced demo `CommunityPage.tsx` and `mocks/community.ts` removed.
- **Pending live verification**: Execute smoke tests against `https://nativityguard.com` and `https://nativityguard-backend.onrender.com/health` (including CORS preflight, auth, unit tabs, and community feed) and record sanitized evidence before marking staging/production release gates as passed.
