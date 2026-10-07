# Live SOS responder locations

This feature tracks an explicitly assigned officer account for each responding unit. `AcceptedBy` remains the administrator who accepted/assigned the unit; it is never used as the GPS owner.

## What ships

- One assigned officer user account per accepted unit, chosen by that unit's active admin/head admin or a super admin.
- Only active officer memberships linked to real user accounts are eligible. Standalone records in the `officers` directory are not login accounts and cannot be tracked.
- The assigned officer enables the existing location-sharing preference in Settings and explicitly starts a per-SOS sharing session.
- Authenticated GPS uploads use the session ID; the server timestamps fixes and stores them in the existing `user_locations` table. The responder links only to a fix uploaded for that SOS session, not the officer's general location history.
- Reporter/authorized staff maps refresh every 10 seconds; fixes older than two minutes are omitted. Client expiry uses server time, so a skewed phone clock does not prolong visibility.
- Stop, reassignment and closure clear the linked fix/session. Disabling location sharing invalidates all that officer's SOS sessions. Old tabs cannot stop a newer session or upload after revocation.
- Hiding/leaving the page stops the browser watcher and requests session revocation. If the tab crashes or the network cannot deliver Stop, the last fix expires within two minutes. This is foreground browser tracking, not background phone tracking.
- GPS permission/connection errors stop uploading; locations are never queued offline. A GPS fix older than 15 seconds on the device is not uploaded as new.
- Citizen responses include unit names, anonymous responder marker IDs and coordinates, not officer names, user IDs, contact details, audit actors or session tokens. Managing admins receive the assigned user ID for their officer selector.
- The detail endpoint returns both `alert` and `sos` aliases plus `responders`; the frontend accepts either envelope.
- Map tile failure preserves the marker list. Closed SOS alerts cannot be reopened through status/release actions.

## API

All routes are under `/api/v1`, require bearer authentication, and return `Cache-Control: no-store` on tracking/detail reads. The existing map/general rate limiters apply.

| Method | Path | Purpose |
|---|---|---|
| GET | `/sos/:id/responders/locations` | `{sosId, serverTime, responders: [{id,unitId,role,latitude,longitude,accuracy,recordedAt}]}`; no current position means `[]`. |
| GET | `/sos/:id/responders/:unitId/officers` | Authorized admin's eligible account list: `{officers:[{userId,firstName,lastName}]}`. |
| PUT | `/sos/:id/responders/:unitId/officer` | `{userId:"UUID"}` assigns; `{userId:null}` clears. Reassignment always revokes previous sharing. |
| POST | `/sos/:id/responders/:unitId/tracking` | Assigned officer explicitly starts; returns `{sessionId}`. |
| PUT | `/sos/:id/responders/:unitId/location` | `{sessionId,latitude,longitude,accuracy}` from that officer. Server timestamps the fix. |
| DELETE | `/sos/:id/responders/:unitId/tracking?sessionId=UUID` | Stops only the matching sharing session. |

Read access: owner, super admin, active admin/head of an addressed/responding unit, or the specifically assigned active officer. An active admin explicitly notified by the existing dispatcher may triage that open SOS. Ordinary unit membership does not grant access. Inaccessible alerts return 404 to avoid leaking their existence.

Existing `/sos`, `/sos/:id` and `/sos/:id/status` now delegate to the same scoped helpers. Status values are validated and terminal states clear sharing. Accept/release require active administration of the relevant unit. The unit-acceptance data model and unit dispatch engine are otherwise retained.

## Schema and rollout

`models.SOSResponder` adds nullable UUID columns `assigned_user_id`, `tracking_session_id` and `location_id`, plus an index on the assigned user. Existing rows remain valid and initially unassigned. No dependency or lockfile changes are needed.

On a dedicated development database, use the existing migration command from `backend/`:

```powershell
go run ./cmd/migrate
```

The migration command uses `DATABASE_URL`. Verify this is a development database before running it. Coordinate the additive schema migration with the deployment owner before deploying code to a shared environment. Do not run the test suite against a production database.

## Verification

```powershell
cd backend
go build ./...
go vet ./...
go test ./...
# TEST_DATABASE_URL must be an isolated disposable database, different from DATABASE_URL.
go test -tags=integration ./handlers -run '^TestSOSTracking' -count=1
cd ..\frontend
npm.cmd ci
npm.cmd run build
npm.cmd test
```

The new integration lifecycle uses real routes/JWT middleware and database queries. It covers accepting a unit, authorization, candidate assignment, response privacy, explicit sharing, stale/invalid positions, delayed uploads after stop, overlapping tab sessions, disabling/re-enabling sharing, reassignment, revoked memberships, closure and malformed IDs. It never invokes the send-SOS endpoint or real emergency notifications.

Manual acceptance test on development data:

1. Use an SOS addressed to the testing unit (or one the dispatcher explicitly notified that admin about). Admin accepts it.
2. Admin opens the SOS detail link and assigns an active officer member.
3. The officer opens the assignment notification, enables location sharing in Settings, returns to the SOS and presses **Start sharing for this SOS**. Allow browser location access.
4. In a separate browser/profile, the reporter opens the same SOS. A fresh green responder marker appears; the orange marker is the SOS.
5. Stop sharing, hide the officer's page, revoke/reassign, disable sharing or close the SOS. Check positions disappear on the reporter's next successful refresh (at most 10 seconds normally); offline clients expire stale positions after two minutes.
6. Check another citizen, an unassigned officer in the same unit and another unit's admin cannot read or publish these locations. Check rejected GPS permission and offline states.

Use HTTPS on phones; localhost works for testing on the same computer. Do not simulate a real emergency on the shared service.

## Boundaries

No background tracking, ETA/routing engine, native app, new map provider, guaranteed emergency dispatch or production deployment is included. Real-device GPS and the team's deployed database still require acceptance testing. Existing acceptance/force-assignment concurrency is outside this patch; this change does not claim to fix the complete dispatch engine. Tracking writers lock the SOS and responder, and stale session tokens fail closed.
