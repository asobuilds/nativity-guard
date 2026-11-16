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
