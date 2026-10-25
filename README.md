# Nativity Guard

> Community safety and governance infrastructure for rural Nigerian communities.

## The Problem

In rural Nigerian communities, security response is hampered by three interconnected crises:

**Delayed emergency response**: SOS alerts often travel through fragmented channels without reliable media evidence. A voice note or short video from the scene can make the difference between a timely response and a missed intervention, but no existing system reliably captures, preserves, and routes this content to the right responders.

**Poor network connectivity**: Many operating communities have limited bandwidth. Large, unoptimized file uploads fail on slow connections, and content that could serve as critical evidence is never transmitted because the upload infrastructure wasn't designed for constrained networks.

**Vigilante fragmentation**: Without a unified platform, local security units operate in isolation. Each community builds its own ad-hoc reporting system, creating parallel ecosystems where information doesn't flow between neighboring units, and where the risk of vigilantism increases because there's no auditable, authorized workflow for case handling.

These problems are not theoretical. They are the daily reality for communities that need rapid, evidence-backed response but lack the infrastructure to support it.

## The Solution

Nativity Guard bridges this gap by providing a unified, auditable workflow that connects citizens, security units, officers, and administrators through a single path: **report ? assign ? dispatch ? investigate ? review ? close**. It is designed to promote accountability, prevent vigilantism, and keep sensitive case intelligence restricted to authorized personnel.

The system introduces three capabilities that directly address the problems above:

**SOS with voice/video**: Alerts can now include voice notes and short video attachments (capped at 5 MB audio, 20 MB video), captured via the device's recorder and uploaded through a content-addressed, size-verified presign-PUT-confirm flow. The content is stored under SHA-256 hashing, so identical footage from different incidents deduplicates automatically — the store holds one copy, and a refcount tracks how many parents reference it.

**Geo-scoped community feeds**: The feed is server-derived, not client-supplied. Place hierarchy follows the five-level ladder decided by the room: locality → LGA → state → country → world. A reply is allowed only inside the viewer's own place at that level. This eliminates the problem of clients sending false `state`/`lga` strings (which have been shown to repeat across jurisdictions) and ensures that geography is always derived from verified coordinates.

**LGA convergence groups**: Within a single (state, LGA) channel, any verified unit member can join or leave, and head admins can post. This creates a proper audit trail for who creates posts in a given LGA, while still allowing any verified unit member to read. The read predicate is strict: a viewer at scope X sees posts whose own scope rank is ≤ X **and** whose place matches the viewer's own place at that level.

## System Features

**SOS Alert with Media**
- Voice notes capped at 5 MB (reuses MaxImageSize; 60 s of Opus/AAC at 64–96 kbps ≈ 0.5–0.7 MB, so 5 MB is ~7–10× headroom)
- Short video capped at 20 MB (1/5 of the evidence 100 MB cap; 30 s at ~5 Mbps ≈ 19 MB, so 20 MB ≈ 30 s of typical phone video)
- Presigned PUT URL flow: caller-scoped `POST /sos/media/presign` (gated by `requireMinorApproved`); client PUTs bytes directly to the bucket; `POST /sos/send` Head-verifies the object's prefix, size, and MIME type before persisting on the SOSAlert row
- MediaLink model with unique index on (object_key, parent_type, parent_id); `OnConflict: DoNothing` makes repeat attach idempotent

**Geo-Scoped Community Feed**
- Five-level place hierarchy enforced server-side: locality → LGA → state → country → world
- Replies gated by place: a user may reply only if the post is visible inside their own place at that level
- LGA convergence groups: any verified unit member can join/leave; head admins may post; read access is per-(state, LGA)
- Server-derived place from coordinates via reverse geocoder; client never sends place name strings

**Media Infrastructure**
- Content-addressed store: identical bytes deduplicate to a single key across different parents
- `MediaLink` model: `objectKey`, `parentType`, `parentId` with unique index `idx_media_link_unique`
- Refcount guard: `removeMediaLink` checks remaining links before deleting the object; deleting the last link triggers storage deletion
- Two private categories: `sos_media/` and `report_media/`
- Serve authorization: `sos_media` requires `?sosId=` + `sosReadScope` check; `report_media` requires `?reportId=` + `postVisibleTo(user, post)` **or** `canModeratePost(user, post, place)`

**Case Lifecycle**
pending → assigned → dispatched → on_scene → investigating → pending_admin_review → closed

Closure requires approval from at least 2 of 2–3 submitted admins. Direct officer closure is blocked.

**Role Hierarchy**
Super Admin (platform-level) → Unit → Head Admin (1 per unit, elected by admins) → Admin (5–10 per unit, elected by verified members) → Officer (tier: primary / paired / support) → Member (verified or provisional)

**Governance**
- Elections: 12-month terms, staggered rotation every 6 months, term limit 2 consecutive terms + 6-month cooling-off, quorum 50% of eligible verified voters
- Revocation: Regular admin removal requires 10+ verified votes + Head Admin approval; Head Admin removal requires majority of verified unit members
- UnitAuth: Per-unit policy document (thresholds, quorum, seat bands, term rules, cooling-off); every cycle snapshots the policy version at open time; no retroactive rule changes

**Security Principles**
- JWT revocation via `jti` checked on every authenticated request
- Refresh token rotation: one-time-use, hash-only storage
- Rate limiting: per-IP and per-identity on auth, OTP, vote, invite
- Password change revokes all sessions immediately
- Tiered case access: primary / paired / support officers; supports see less
- Public-safe DTOs: reporter identity, officer identity, evidence internals never leak
- Suspect self-view: active cases on profile, resolved cases in history

**Production Administration**
The live administration surfaces are organized around real backend capabilities. Super admins have platform oversight, user/role management, identity and unit verification, feedback/support, audit, analytics, operations, Community management, and unit records. Unit admins have live case review, transfers, roster/unit operations, Community, analytics, settings, governance, map, appeals, and feedback surfaces.

Community reads and writes are live. The feed is scoped server-side to the caller's own place (locality → LGA → state → country → world), and a reply is allowed only inside that scope, so geography and authorization are backend rules rather than frontend flags. The former `VITE_COMMUNITY_WRITES_ENABLED` gate has been retired.

## Technical Approach

**Offline-first design**: The system is built to function without constant connectivity. Where possible, operations complete on the device before syncing; where server coordination is required, the flow is minimized to reduce round trips.

**Server-derived geolocation**: The platform never trusts client-supplied place names. Place is always derived from verified coordinates via reverse geocoder (decisions D-011…D-016 in resolution.md). This prevents the spoofing vulnerability where a client could claim to be in a different LGA or state than they actually are.

**Data deduplication**: The store is content-addressed. Identical bytes under the same category share one key across different parents. The `MediaLink` refcount guards against accidental GC: deleting one parent's link does not remove the key if another parent still references it. This is essential for the SOS/report use case where the same voice note or video might be attached to multiple alerts or reports.

**Go backend**: Gin routing, GORM PostgreSQL ORM, JWT authentication with `jti`-based revocation, bcrypt password hashing, GORM auto-migration, migration-based schema evolution.

**React frontend**: React 19, TypeScript, Vite, Tailwind CSS, React Query for data fetching, Leaflet for map rendering, responsive AppShell drawer at `lg` breakpoint with five theme palettes and carved-mask artwork.

**Infra**: Docker Compose for local development; Render for backend+frontend hosting; Supabase PostgreSQL for the production database; env variables stored outside the repository (never committed).

**Authorization tiered**: Per-IP and per-identity rate limiting on auth, OTP, vote, and invite endpoints; JWT revocation list checked on every request; refresh token rotation (one-time-use); password change revokes all existing sessions; tiered case access controls (primary/paired/support officers see only their assigned cases).

## Getting Started

### Backend
```bash
cd backend
go mod download
go run ./cmd/migrate
go run ./cmd/api      # or: go run main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

### Environment
Create `backend/.env`:
```
DATABASE_URL=postgresql://...
JWT_SECRET=<long-random-string>
PORT=8080
```

Never commit `.env`. Rotate secrets if ever leaked.

## Contributing
- One feature per branch
- Build + vet must pass before PR
- All auth changes require a second pair of eyes
- Docs live alongside code — update README and AGENT.md in the same commit

## License
Private — all rights reserved.

## Live SOS Responder Tracking
The SOS detail screen supports explicit officer assignment and foreground location sharing. See the responder-tracking guide for permissions, additive migration, API contracts and verification steps.

### Unit Financial Ledger & Performance Caching
- **Bank Account Encryption**: AES-GCM encryption of bank account numbers via `cryptoutil.Encrypt()` on save and automatic decryption on read.
- **Webhook Idempotency & Donation Processing**: Idempotent payment webhooks (`WebhookEvent`) creating immutable `FinancialLedger` entries (`Direction="in"`, `EntryType="donation"`).
- **Balance & Transactions APIs**: `GET /units/:id/ledger/balance` and `GET /units/:id/transactions` (pending transactions).
- **Alive Map Routing & Custom Landmarks**: OSRM routing integration with polyline rendering, floating HUD ETA badges, and unit custom landmarks ("Safe House", "Known Hotspot", "Checkpoint").
- **Web Push Notifications**: Real-time browser push notifications via service worker (`sw.js`) and VAPID webpush-go dispatching.
- **Redis Performance Caching**: Cache-aside pattern for community feeds, unit ledger balances, and map POIs with robust graceful fallback when Redis is offline.

### Responsive Navigation and Public Artwork (October 2026)
Authenticated navigation uses a dismissible slide-out drawer below the `lg` breakpoint; the close (X) control, menu button, backdrop, and bottom navigation share that breakpoint. At `lg` and wider, the sidebar remains docked. The carved-mask artwork on public landing/onboarding pages and all five selectable themes are retained. PR #63 addresses the responsive navigation regression; browser-based device/role acceptance testing is still required and CI success alone does not establish visual correctness.

### Authentication Artwork Parity
Login and signup use the same `ng-public-scene` theme-aware carved-mask background treatment as the landing and onboarding pages, on their desktop branding panels. Forms, account validation and existing responsive breakpoints remain unchanged. Verify visually on desktop and mobile before calling the layout fixed.

---