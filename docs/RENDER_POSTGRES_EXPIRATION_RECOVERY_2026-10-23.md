# P0: Recover Nativity Guard PostgreSQL after Free-plan expiry

**Incident identified:** 2026-10-23  
**Database:** `nativityguard-db` (existing Render Postgres instance)  
**Affected service:** `nativityguard-backend`  
**Status at discovery:** Free Postgres `suspended`; backend restart-loop with database DNS lookup failures  
**Severity:** P0 — public API unavailable and account, legal-document, safety-report and tracking workflows may be broken

## Evidence

- Render's database status is `suspended` with suspender `billing`; plan `free`.
- Database expiration: 2026-10-23 06:35:26 UTC.
- Repeated Render backend `server_failed` events during 2026-10-23 around 10:12–10:18 UTC.
- Backend logs show database connection startup failing because the internal database hostname is not resolvable. Do not publish credentials or the full connection URL in a ticket.
- Live Chrome checks: homepage, public map pop-up and About page work; Terms/Privacy page remains in `Loading…` longer than 20 seconds. Both policy requests start but do not settle while the backend is unavailable.

## Data-preserving recovery: existing database only

1. Sign in to the existing Render workspace.
2. Open the existing database, not a new empty instance:
   https://dashboard.render.com/d/dpg-dapn6bg473hc73cb8vq0-a
3. In its **Plan** or compute-plan settings, select a supported paid Postgres plan and review the price before approving the change.
4. Verify that the **existing** database returns to `available` / `active` status. Do not delete it or rotate connection credentials as a first troubleshooting step.
5. Confirm the backend can connect again and its `/health` endpoint responds. Only after the database is available should a service restart/redeploy be considered if needed.
6. Verify both live public policy endpoints, an authorised login flow and read-only case/unit summaries. Avoid submitting real emergency SOS or test donations.
7. Make a tested backup/export after recovery, and document retention and monitoring.

**Do not create a replacement Free database as a shortcut.** A replacement starts empty and does not recover current user data.

## Time limit

Render's official Free Postgres policy: expiry after 30 days, followed by **14 days to upgrade**. After the grace period, the instance and data are deleted. On the published expiry timestamp, this suggests a recovery deadline around **2026-11-06 06:35 UTC**. This date is calculated from published policy; confirm the exact deadline in Render before relying on it.

References:
- https://render.com/docs/free
- https://render.com/changelog/free-postgresql-instances-now-expire-after-30-days-previously-90

## Tool/authority limits

The installed Render connector supports listing and inspecting Postgres, querying accessible databases, creating new ones, and deploying web services, but it does **not** expose a change-compute-plan action. It cannot rescue an expired suspended instance from this chat.

The recovery may require payment. No paid upgrade, database mutation, secret change or production redeploy has been performed during this investigation.

## Acceptance gate

- [ ] Existing DB status is available (not suspended)
- [ ] Backend starts without DNS/database errors and health check passes
- [ ] Production Terms and Privacy documents finish loading with real versioned content
- [ ] Existing records remain present; data integrity checked
- [ ] Safe backup/export created and recovery tested
- [ ] Functional browser and mobile checks rerun; only then reconsider release PR merges
