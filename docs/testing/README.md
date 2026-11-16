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

- [ ] `backend-ci` passes on the exact release commit.
- [ ] `frontend-ci` passes on the exact release commit.
- [ ] `integration-race` passes on the exact release commit.
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
