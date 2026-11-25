# Nativity Guard — safe public backend outage notice

**Incident:** NG-25 — Render Free PostgreSQL expiry and backend startup failures (2026-10-23).

## What changed

The public landing page retains its existing City Gate design and primary actions. A compact notice appears **only** when the real backend's database-aware `GET /health` endpoint reports a server/database failure or cannot be reached before a bounded 8-second timeout.

It warns that sign-in, reports, tracking, and in-app SOS may not work and provides a direct `tel:112` link for urgent help in Nigeria. It does not pretend that emergency dispatch has occurred.

The probe uses the configured backend API origin. It checks the official `status` plus `checks.database.ok` fields; an unrelated HTML page or a 404 response cannot be treated as healthy or proof of an outage. Response values other than clear success/failure remain unknown and do **not** trigger a warning.

The public probe is unauthenticated and sends no credentials, position data, account details, or case content. It runs at most once every two minutes per actively open landing page, plus focus-triggered checks, and visitors can select **Check again**.

## Production and release gates

- [ ] Existing Render database upgraded/restored rather than replaced.
- [ ] Backend connects and `/health` reports database healthy.
- [ ] Check banner appears while intentionally stubbed health is unavailable.
- [ ] Check banner disappears when the actual health endpoint recovers.
- [ ] Confirm small-screen, keyboard, and screen-reader appearance.
- [ ] Confirm legal docs, reporting, and sign-in work after database recovery.
- [ ] Review and merge only after feature acceptance; tests alone do not repair the database.

**Draft-only UI change.** It cannot itself revive the suspended database or guarantee emergency assistance.
