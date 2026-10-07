# Nativity Guard

> Community safety and governance infrastructure for Nigerian neighborhoods.

Nativity Guard connects citizens, security units, officers, and administrators through one auditable workflow: **report ? assign ? dispatch ? investigate ? review ? close**. It is designed to promote accountability, prevent vigilantism, and keep sensitive case intelligence restricted to authorized personnel.

---

## Status

Backend: **production-ready for governance features**, actively hardening.
Frontend: actively developed in a parallel track.
Deployed on: Render (backend) + Supabase (Postgres).

---

## Core principles

1. **Unit access is NOT case access.** A police unit does not see every case in its jurisdiction — only the cases assigned to specific officers.
2. **Every hierarchy is auditable.** Head admins, admins, and officers are all removable by defined thresholds and quorum.
3. **Nothing auto-deletes without user consent** — except security tombstones, which are documented separately.
4. **No vigilantism.** The platform coordinates, it does not arm.
5. **Presumed innocence.** Suspects see only that a case names them, never the investigative detail.

---

## Architecture

    Citizen  ?  Officer  ?  Admin  ?  Head Admin  ?  Super Admin
       ?           ?         ?           ?              ?
       ?????? Case workflow (assign ? dispatch ? investigate ? review ? close)
                    ?
              CaseAdminAssignment (2–3 admins per case)
                    ?
              RevocationCycle / RevocationVote (removal thresholds)

Backend: Go 1.24 · Gin · GORM · PostgreSQL · JWT · bcrypt
Frontend: React 19 · TypeScript · Vite · Tailwind · React Query · Leaflet
Infra: Docker Compose · Render · Supabase

---

## Case lifecycle

    pending ? assigned ? dispatched ? on_scene ? investigating
       ? pending_admin_review
       ? [admin_changes_requested ? investigating]
       ? closed

Closure requires approval from at least **2 of the 2–3 submitted admins**. Direct officer closure is blocked.

---

## Governance

### Elections
- Admins elected by verified members, top 5–10 per unit
- Terms: 12 months · staggered rotation every 6 months
- Term limit: 2 consecutive terms · 6-month cooling-off
- Head Admin elected by admins, one per unit

### Revocation
- Regular admin removal: **10+ verified member votes + Head Admin approval**
- Head Admin removal: **majority of verified unit members**
- One vote per member per cycle · immutable vote records
- Successful removal triggers cooling-off

### UnitAuth
Per-unit policy document declaring thresholds, quorum, seat bands, and term rules. Every election and revocation cycle snapshots the policy version it was governed by — no retroactive rule changes.

---

## Security

- **JWT revocation** — tokens carry a `jti`; revocation list checked on every authenticated request
- **Refresh token rotation** — one-time-use, hash-only storage
- **Rate limiting** — per-IP and per-identity on auth, OTP, vote, invite
- **Password change** revokes all sessions immediately
- **Tiered case access** — primary / paired / support officers; supports see less
- **Public-safe DTOs** — reporter identity, officer identity, evidence internals never leak
- **Invite scoping** — platform invites (anyone) vs unit invites (admin/head only, post-verification)
- **Suspect self-view** — active cases on profile, resolved cases in history

---

## Roadmap

| Wave | Focus | Status |
|---|---|---|
| 1 | Secrets audit + `.gitignore` hardening | ? |
| 2 | Auth hardening (rate limits, JWT revocation, refresh) | ? |
| R | Rebrand to Nativity Guard | In progress |
| 2b | Session persistence (device list, per-device revoke) | Next |
| 3 | File & data safety (upload validation, MedicalInfo encryption) | Planned |
| 4 | Data integrity (transactions, idempotency, scheduler lock) | Planned |
| 4b | Universal authority matrix | Planned |
| 5 | Justice & ethics (secret ballot, right to respond, expungement, appeal) | Planned |
| 5b | Data retention + user-initiated delete flow | Planned |
| 6 | Ops resilience (pagination, structured logs, monitoring) | Planned |
| 7 | Governance strengthening (quorum fix, tie-break, vacancy fill) | Planned |
| 8 | Maturity (event bus, versioned migrations, tests, compliance) | Planned |

---

## Getting started

### Backend
    cd backend
    go mod download
    go run ./cmd/migrate
    go run ./cmd/api      # or: go run main.go

### Frontend
    cd frontend
    npm install
    npm run dev

### Environment
Create `backend/.env`:
    DATABASE_URL=postgresql://...
    JWT_SECRET=<long-random-string>
    PORT=8080

Never commit `.env`. Rotate secrets if ever leaked.

---

## Contributing

- One feature per branch
- Build + vet must pass before PR
- All auth changes require a second pair of eyes
- Docs live alongside code — update README and AGENT.md in the same commit

---

## License

Private — all rights reserved.


## Live SOS responder tracking

The SOS detail screen supports explicit officer assignment and foreground location sharing. See [the responder-tracking guide](docs/teammates/live-responder-locations.md) for permissions, additive migration, API contracts and verification steps.
