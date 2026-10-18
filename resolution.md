# resolution.md — Action Ledger

> What the room (`agentmeeting.md`) **agreed**, and what is **actually moving**,
> with timeboxes and evidence. Append-only except for the status fields of a row
> you own. Contract: [`AGENT.md`](./AGENT.md).

All timestamps UTC ISO-8601. Status values:

| Status | Meaning |
|---|---|
| `unclaimed` | Free to take. Claim before editing code. |
| `claimed` | Owner has announced it in `agentmeeting.md` Section 6. |
| `in_progress` | Code being written; branch exists and is pushed. |
| `in_review` | PR open; waiting on review/merge. |
| `blocked` | Waiting on another row (`after: RES-xxx`) or a human decision. |
| `done` | Merged to `main` **and** evidence recorded below. |
| `dropped` | Abandoned; reason recorded. |

---

## Cycle

| Field | Value |
|---|---|
| Cycle | `CYCLE-2026-10-17` |
| Opens | 2026-10-17T17:00:00Z |
| Review checkpoint | 2026-10-20T17:00:00Z |
| Closes | 2026-10-24T17:00:00Z |
| Goal | Clear the security hardening queue without regressing the navigation/theme work shipped in PR #63. |

---

## Active board

| ID | Workstream | Feature | Owner | Branch | Files / area | Status | Started | PR |
|---|---|---|---|---|---|---|---|---|
| RES-000 | Meta | Multi-agent coordination protocol | `claude` | `main` (uncommitted) | `agentmeeting.md`, `resolution.md` | `in_review` | 2026-10-17T16:03Z | — |
| RES-001 | Security | Global settings authorization | `claude` | `fix/super-admin-global-settings` | `backend/handlers`, `backend/middleware`, unit-admin UI | `done` | — | #62 |
| RES-002 | Security | Last-super-admin protection | `unclaimed` | `fix/last-super-admin-protection` | `backend/handlers` (user mutation), `backend/models` | `unclaimed` | — | — |
| RES-003 | Security | Final role-by-role authorization audit | `unclaimed` | `fix/role-route-authz-audit` | `backend/routes/routes.go`, `backend/middleware` | `unclaimed` | — | — |
| RES-004 | Community | RSVP concurrency / capacity safety | `opencode` | `fix/community-rsvp-concurrency` | `backend/handlers/community_handler.go` (RSVP path), `backend/models/Community.go`, `backend/config/migrate.go`, `backend/internal/testutil/setup.go`, new `backend/handlers/rsvp_concurrency_test.go` | `done` | 2026-10-17T16:12Z | #69 |
| RES-005 | Community | Real Community moderation + geo-scoped feed (locality → LGA → state → country → world) | `opencode` | `feat/community-geo-scoped-feed` | `backend/models/Community.go`, `backend/handlers/community_handler.go`, new `backend/handlers/community_geo.go`, `backend/routes/routes.go` (append only), `frontend/src/pages/LiveCommunityPage.tsx`, new `frontend/src/components/community/ScopeSelector.tsx` | `done` | 2026-10-18T12:29Z | #71 |
| RES-006 | Finance | Finance/bank route authorization, atomic approvals, API contract alignment | `antigravity` | `fix/finance-route-review` | `backend/handlers/{finance,bank_account,ledger,public_bank,platform_donation}_handler.go`, `backend/routes/routes.go` (aliases only), new `backend/handlers/finance_auth_test.go` | `claimed` | 2026-10-17T16:19Z | — |
| RES-007 | Docs | Reconcile stale hardening queue in README/AGENT.md | `unclaimed` | `docs/reconcile-hardening-queue` | `README.md`, `AGENT.md` | `unclaimed` | — | — |
| RES-008 | Security | Event RSVP object-level authorization (cross-unit + non-upcoming) | `unclaimed` | `fix/community-rsvp-authorization` | `backend/handlers/community_handler.go` (`RSVPToEvent`) | `unclaimed` (`RES-004` merged — dependency cleared) | — | — |
| RES-009 | Frontend | Perceived reliability: global error surfacing, loading states, mutation feedback | `claude` | `feat/frontend-reliability-pass` | `frontend/src/lib/queryClient.ts`, `frontend/src/components/ui/States.tsx`, data-heavy pages | `claimed` | 2026-10-17T16:27Z | — |
| RES-010 | Infra | Backend build/vet CI gate (none exists today) | `unclaimed` | `ci/backend-build-vet` | `.github/workflows/` | `unclaimed` | — | — |
| RES-011 | Frontend | Phone drawer width/insets + public background legibility | `opencode` | `fix/responsive-drawer-and-public-backgrounds` | `frontend/src/components/layout/AppShell.tsx`, `frontend/src/index.css`, `frontend/src/theme.css`, new `frontend/src/components/layout/shellInsets.test.ts`, `frontend/package.json` (adds `@types/node@^22`), renamed `frontend/public/images/nativity-carved-masks.webp` | `done` | 2026-10-18T10:20Z | #67 |
| RES-012 | Community | LGA convergence group: join/leave/roster, head-admin-only posting | `opencode` | `feat/community-geo-scoped-feed` | new `backend/models/LGAGroupMember.go`, `backend/handlers/lga_channel_handler.go`, `backend/routes/routes.go` (append only), `frontend/src/components/unit/LGAChannelPanel.tsx` | `done` | 2026-10-18T12:29Z | #71 |
| RES-013 | Docs | Community geo-feed research + architecture document | `opencode` | `feat/community-geo-scoped-feed` | new `docs/community/GEO-SCOPED-FEED.md` | `done` | 2026-10-18T12:29Z | #71 |
| RES-014 | Community | Report + moderation queue endpoint; keyset pagination; `view_count` atomicity | `opencode` | `fix/community-report-queue-pagination` | new `backend/handlers/community_moderation.go`, `backend/handlers/community_handler.go` (`GetForumPosts`, `GetForumPostByID`), new `backend/handlers/community_moderation_test.go`, `backend/config/migrate.go`, `backend/routes/routes.go` (append only), `frontend/src/pages/LiveCommunityPage.tsx` | `done` | 2026-10-18T13:38Z | #74 |
| RES-015 | Community | Deferred community hardening: content takedown/hide, ranking, scope-aware notifications, dead `ChatMessage` model, "communication room" membership authz scan, unrouted demo `CommunityPage.tsx` + its mock | `unclaimed` | `feat/community-hardening` | `backend/handlers`, `backend/models`, `frontend/src/pages/LiveCommunityPage.tsx`, `frontend/src/pages/CommunityPage.tsx`, `frontend/src/mocks/community.ts` | `unclaimed` | — | — |
| RES-016 | Community | Community feature audit: Discussions, Announcements, Events — integration flow and phantom endpoints (recon only, no code) | `opencode` | `docs/community-feature-audit` | new `docs/community/COMMUNITY-FEATURE-AUDIT.md`, plus `resolution.md` / `agentmeeting.md` ledger rows only | `in_review` | 2026-10-18T16:13Z | #77 |
| RES-017 | Community | Media attachments on report and SOS: existing infrastructure recon + implementation plan (recon only, no code) | `opencode` | `recon/media-attachments-report-sos` (stacked on `docs/community-feature-audit` / #77 — merge #77 first) | new `docs/community/MEDIA-ATTACHMENTS-PLAN.md`, plus `resolution.md` / `agentmeeting.md` ledger rows only | `in_review` | 2026-10-18T16:32Z | #78 |
| RES-020 | Community | Media attachments on report and SOS with refcount (RES-020 implementation) | `opencode` | `feat/media-attachments-sos-report` | new `backend/handlers/media_links.go`, `backend/handlers/community_media_handler.go`, `backend/handlers/sos_media_handler.go`, `backend/models/MediaLink.go`, `backend/routes/routes.go`, `backend/config/migrate.go`, new `backend/handlers/media_links_test.go`, `backend/handlers/media_presign_sos_test.go`, `backend/handlers/media_send_sos_test.go`, `backend/handlers/media_serve_sos_test.go`, `backend/handlers/media_presign_report_test.go`, `backend/handlers/media_report_attach_test.go`, `backend/handlers/media_serve_report_test.go`, `backend/handlers/media_filestorage_test.go`, `backend/handlers/media_dedup_test.go` | `in_progress` | 2026-10-18T21:45Z | — |
| RES-018 | Community | Announcements and Events geo-scope: product decision required | `unclaimed` | — | `resolution.md` (ledger row + decision brief only — **no code**) | `unclaimed` | — | — |
| RES-019 | Community | Moderation notification on report creation | `opencode` | `fix/report-moderation-notification` | `backend/handlers/community_moderation.go`, `backend/handlers/sos_handler.go` (`notifyUser` gains a type param), new `backend/handlers/community_report_notification_test.go`, `backend/internal/testutil/setup.go` (test table list only) | `in_review` | 2026-10-18T17:16Z | #79 |

**Suggested split for the three agents** (a suggestion is not a claim — confirm
in `agentmeeting.md` before starting):

- `claude` — RES-002, then RES-003 (backend authorization, needs the broadest
  repo context).
- `antigravity` — RES-005 frontend half only once a backend contract exists;
  until then RES-006 review (read-mostly, no hot files).
- `opencode` — RES-004 (self-contained handler + migration), then RES-007.

---

### RES-018 — decision brief: Announcements and Events geo-scope

Filed from `docs/community/COMMUNITY-FEATURE-AUDIT.md` findings 2 and 3. **No code is
written and no scope is chosen here — this row is for the owner to decide.**

Findings (evidence in the audit):
- `CommunityAnnouncement` and `CommunityEvent` have **no `GeoPlace`** embed and their read
  filters are `unit_id`-only, so each can address exactly one unit or everyone — never an
  LGA or state, and never the viewer's own place (`models/Community.go:79-127`;
  `community_handler.go:551-561`, `:691-701`).
- Events therefore **contradict D-015**: the LGA convergence group deliberately spans units
  within one LGA, while an event cannot reach a neighbouring unit in the same LGA. The two
  features disagree on what "local" means.

Options — the owner picks one:
- **(a)** Add a `GeoPlace` embed to both models and reuse `scopeWindowCondition` /
  `placeVisibleToViewer`, giving announcements and events the same
  `locality → lga → state → country → world` ladder as forum posts. Largest change; it
  changes who sees existing rows.
- **(b)** Keep the unit-only filter and **document the divergence** from D-011/D-013 as
  deliberate (announcements/events are unit-scoped by design).
- **(c)** Introduce a new **LGA-level audience** that spans units, aligning events with
  D-015's LGA group.

**Not decided. No implementation.** If (a) or (c) is chosen, it becomes its own row.

---

## Decisions

| ID | Date | Decision | Rationale | Decided by |
|---|---|---|---|---|
| D-001 | 2026-10-17 | One feature = one owner = one branch; claims live in this file. | Two agents editing one handler causes silent authz drift. | `claude` |
| D-002 | 2026-10-17 | Hot files (Section 4 of `agentmeeting.md`) require a meeting entry even when owned. | `AppShell.tsx` and `routes.go` are the two highest-collision files in the repo. | `claude` |
| D-003 | 2026-10-17 | Log appends go at the bottom of a section, under the agent's own heading. | End-appends merge cleanly; interleaved edits conflict. | `claude` |
| D-004 | 2026-10-17 | `global settings authorization` is treated as complete (PR #62) and removed from the active queue. | Code and merge history supersede the stale README line. | `claude` |
| D-005 | 2026-10-17 | Nav/theme work (PR #63) is frozen unless the human owner requests otherwise. | Explicit regression guard appended to `AGENT.md`. | `claude` |
| D-006 | 2026-10-17 | Each agent works in its **own git worktree**, not the shared directory: `claude` → `/home/asobuild/nativity-guard`, `antigravity` → `/home/asobuild/nativity-guard-ag`, `opencode` → `/home/asobuild/nativity-guard-oc`. | Three processes sharing one checkout means any `git checkout -b` switches branches under the other two. Worktrees share `.git` and `origin` but have separate working trees, so `fetch` sees every claim and `checkout` affects only its own tree. Resolves OPF-001's second question. | human owner + `claude` |
| D-007 | 2026-10-17 | Approved the RES-004 / RES-005 split: RES-004 owns the RSVP path (`RSVPToEvent`, `EventAttendee`, `CommunityEvent.AttendeeCount`); RES-005 owns moderation and `LiveCommunityPage.tsx`. | Resolves OPF-002. Same file, different functions, so the split is by function and enforced by review, not by path. | `claude`, for the owner |
| D-008 | 2026-10-17 | New timestamps are machine UTC (`date -u +%Y-%m-%dT%H:%M:%SZ`). Existing entries are append-only and stay as written. | Resolves OPF-003. Re-stamping history would violate §2.6; the ladder compares claim order, so the skew is cosmetic. | `claude` |
| D-009 | 2026-10-17 | RES-006 is scoped to finance/bank handlers and finance tests. RES-003 (route-wide authz audit) remains a separate, later pass. | Antigravity's claim is broad; explicit scope prevents RES-003 becoming a duplicate. | `claude` |
| D-010 | 2026-10-17 | RES-006 may alias `/bank/units/:id/accounts` in `routes.go`, but must not re-register or regroup existing routes. | `routes.go` is hot; append-only registration keeps RES-003/RES-006 merges clean. | `claude` |
| D-011 | 2026-10-18 | Community geography is a five-level scope ladder, most specific first: `locality` (village/ward/neighbourhood) → `lga` → `state` → `country` → `world`. | Matches the owner's requirement exactly. `locality` is the honest finest level: `data/nigeria.json` has states and 774 LGAs but **no wards or villages**, so the village name comes from the reverse geocoder's `Suburb`/`Village`/`Hamlet`, or from `SecurityUnit.Ward` for staff. | `opencode`, for the owner |
| D-012 | 2026-10-18 | A post's place is **server-derived** from coordinates (reverse-geocoded, cached) or the author's verified unit. A client may send `scope` and `lat`/`lng`; it may **never** send `state`/`lga`/`locality` strings. | Same posture as the existing `callerLGAContext`, which already refuses to take LGA/state from the request. Trusting client place names is a spoofing hole in a jurisdiction-scoped feed. | `opencode`, for the owner |
| D-013 | 2026-10-18 | The feed is a **strict audience window**: a viewer at scope X sees posts whose own scope rank is ≤ X **and** whose place matches the viewer's own place at that level. An `lga` match is always paired with `state`. | LGA names repeat across states, so `lga` alone is ambiguous. The window can widen but never move — a Lagos viewer cannot widen into Benue. | `opencode`, for the owner |
| D-014 | 2026-10-18 | **Visibility = participation.** You may reply to a post only if it is visible to you (same predicate). Reply, detail view and feed all share one function. | Directly implements "users cannot comment on things of other places" with a single rule, so the read boundary and the write boundary cannot drift apart. | `opencode`, for the owner |
| D-015 | 2026-10-18 | The LGA convergence group keeps the existing per-`(state, lga)` channel for **read** by any verified unit member in that LGA, adds explicit join/leave + a unit roster, and restricts **posting** to head admins (`UnitMembership.IsHeadAdmin`). | The requirement is "all units in the LGA can see; only head admins create posts". The read half already works; the delta is membership + the posting gate. | `opencode`, for the owner |
| D-016 | 2026-10-18 | Go live: remove the `VITE_COMMUNITY_WRITES_ENABLED` gate and the "being reviewed" banner. Community writes ship enabled once the server enforces scope and authorization. No demo/placeholder data. | The env gate existed because the backend had no scope or moderation boundary (`LiveCommunityPage.tsx:17`). With D-013/D-014/D-015 enforced server-side, the gate is what makes the feature a placeholder. | `opencode`, for the owner |
| D-017 | 2026-10-18 | **OPF-004 closed — Option 1.** `opencode` owns `frontend/src/components/layout/AppShell.tsx`; `claude` discards its uncommitted copy in `/home/asobuild/nativity-guard` and keeps `Toast.tsx`/`queryClient.ts` for RES-009. | Resolves the two-writer collision OPF-004 raised. The ruling is also the safer one on the merits: RES-011 already merged (#67, `31eb19d`), so `main` holds the drawer cap and the five real `safe-*` inset definitions; claude's stale `3b76787` copy would revert the cap to `max-w-[85vw]` and re-add four undefined class names. | the human owner |

---

## Handoffs

Record partial work that another agent must continue. A handoff is a contract:
state what is done, what is not, and what must be read first.

_None yet._

Hand-off for RES-004 (the branch is committed locally but this box cannot push).

### HANDOFF-2026-10-18-01 — from `opencode` to human owner
- Feature: RES-004
- Branch / PR: `fix/community-rsvp-concurrency` (committed locally; **not pushed** — no credentials)
- Done: `RSVPToEvent` is transactional and capacity/duplicate safe; partial unique index
  `idx_event_attendee_active`; two integration tests; local build/vet/unit and
  `-tags integration` type-check all green.
- Not done: `git push` + PR (needs a GitHub credential); `TestRSVP` execution (needs the CI
  Postgres); production duplicate pre-check (needs a read-only `DATABASE_URL`).
- Must read before continuing: this entry; the RES-004 row and evidence above.
- Known risks: `origin/main` still does **not** contain the coordination ledger — this branch
  carries a copy bootstrapped from the untracked working tree. If a separate
  `docs/agent-coordination-protocol` bootstrap is pushed as well, expect an add/add conflict
  on `agentmeeting.md`/`resolution.md`; merge the ledger once. Production `AutoMigrate` will
  fail if duplicate live `(event_id,user_id)` rows exist, and deduping needs owner approval.
- Update 2026-10-18T11:45Z: unblocked — the owner supplied a GitHub credential. The branch was
  merged with `origin/main` (`31eb19d`, which carries the RES-011 changes and the first
  committed copy of this ledger; resolver took `main`'s ledger verbatim, no branch-only
  ledger content was dropped). Backend re-verified after the merge: `go build ./...`,
  `go vet ./...`, `go test ./...` all pass. Branch pushed and PR **#69** opened and merged as
  `9c1cf4f`; the push/PR item above is closed. `TestRSVP` ran in CI and passed (see the
  evidence row). Remaining item: the production duplicate pre-check needs a read-only
  `DATABASE_URL`.

### HANDOFF-2026-10-18-02 — from `opencode` to human owner
- Feature: RES-011
- Branch / PR: `fix/responsive-drawer-and-public-backgrounds` → PR **#67** (pushed 2026-10-18)
- Done: the below-`lg` drawer is capped at `w-[17rem]` / `max-w-[calc(100vw-3.5rem)]` so it always leaves a 56px gutter instead of covering 80% of a phone; the shell clears the iPhone home indicator and Android gesture bar via five `safe-*` classes defined in `index.css` with `env(safe-area-inset-*)` (header, drawer, `<main>`, bottom bar, SOS) plus `lg` overrides; the public-scene artwork is blurred and de-contrasted per theme (`theme.css`) so headings measure >= 12:1 contrast; `shellInsets.test.ts` guards the class-to-rule linkage; a pre-existing minifier/404 bug in the artwork URL is fixed by renaming the asset. Full commands, numbers and the browser-verification method are in the RES-011 evidence row.
- Not done: `git push` + PR (needs a GitHub credential). Real-device safe-area rendering (CDP cannot emulate `env(safe-area-inset-*)`; the mechanism is proven only with `env()`=0). Body scroll is not locked while the drawer is open — pre-existing on `main`, left alone to keep the diff surgical.
- Update 2026-10-18T11:05Z: unblocked — the owner supplied a GitHub credential, `gh auth setup-git` is configured, the branch is pushed and PR #67 is open. The push/PR item in "Not done" is closed; the other two items stand.
- Must read before continuing: this entry; `MEET-2026-10-18-02` (the hot-file huddle) and `MEET-2026-10-18-03` (the report) in `agentmeeting.md`; the RES-011 evidence row.
- Known risks: **OPF-004 is unanswered** — `claude`'s uncommitted working-tree `AppShell.tsx` (blob `3b76787`) touches the same file and would widen the drawer to `max-w-[85vw]` and still ship four dead `safe-*` names. Whoever commits first must reconcile; do not merge both. The branch carries its own copy of `agentmeeting.md`/`resolution.md` (neither is on `origin/main`), so expect an add/add conflict on those two files if another ledger copy is pushed; merge the ledger once. The asset rename assumes nothing else links the old filename — only `/favicon.svg` and `/manifest.webmanifest` are precached by the service worker, so a stale cached `index.html` is the only way to 404.

Template:

```
### HANDOFF-YYYY-MM-DD-NN — from <handle> to <handle>
- Feature: RES-xxx
- Branch / PR:
- Done:
- Not done:
- Must read before continuing:
- Known risks:
```

---

## Evidence log

`done` is not a feeling. Attach the commands and results.

| ID | Date | Agent | Evidence |
|---|---|---|---|
| RES-001 | — | `claude` | PR #62 merged (`ed8c135`, `53c7cbe`, `d7df697`). Backend scoped global settings/templates to super_admin via existing role middleware. |
| RES-000 | 2026-10-17 | `claude` | Created `agentmeeting.md` + `resolution.md`. Not committed pending owner instruction. |
| RES-004 | 2026-10-18 | `opencode` | Local: `go build ./...`, `go vet ./...`, `go test ./...`, `go vet -tags integration ./...` all pass. `TestRSVP` (`rsvp_concurrency_test.go`) executes in CI `integration-race.yml` against `postgres:16`; this box has no Postgres server. Push/PR not run here: no GitHub credentials (gh logged out; no token/SSH). See `HANDOFF-2026-10-18-01`. Re-verified 2026-10-18 after merging `origin/main` (`31eb19d`) into the branch: `go build ./...`, `go vet ./...`, `go test ./...` all pass. **CI proof:** PR #69 `race` job ran the integration suite against `postgres:16` — `TestRSVPConcurrentRespectsCapacity` PASS (32.77s), `TestRSVPConcurrentDuplicateUserSingleRow` PASS (2.19s). **Merged** as `9c1cf4f` (PR #69, 2026-10-18). Remaining item: the production duplicate pre-check needs a read-only `DATABASE_URL`. |
| RES-011 | 2026-10-18 | `opencode` | **Static gate** (`frontend/`, exit 0 each): `npm run typecheck`; `npm run lint` 0 errors / 3 pre-existing warnings (`useOfflineQueue.ts`, `ProfilePage.tsx`, `ReportIncidentPage.tsx`); `npm test` 20 files / **185 passed** (new `shellInsets.test.ts` = 7, pins each shell `safe-*` class to a real `env(safe-area-inset-*)` rule and pins the drawer cap); `npm run build` OK. **Built CSS proof** (`dist/assets/index-*.css`): all five rules survive minification — `safe-top-inset{padding-top:calc(.75rem + env(safe-area-inset-top,0px))}`, `safe-inset-y`, `safe-main-inset{padding-bottom:calc(5rem + env(safe-area-inset-bottom,0px))}` becoming `0` at `@media (min-width:64rem)`, `safe-bottom-bar{padding-bottom:calc(.375rem + env(safe-area-inset-bottom,0px))}`, `safe-sos-offset{bottom:calc(4rem + env(safe-area-inset-bottom,0px))}` becoming `1.25rem` at `lg`; artwork resolves as `url(/images/nativity-carved-masks.webp)` with the file present on disk, 0 occurrences of `%20`/stray quote. **Browser verification** (headless Chrome 155 + CDP against `vite preview` of the dev-mock build; no CI signal used for UI): drawer at 360x740 = w 272px, gutter 88px (share 75.6%); 320x568 = w 264, gutter 56; 740x360 = w 272, gutter 468; 768x1024 = w 272, gutter 496, nav not overflowing; 1280 = docked `lg:w-60` (240px), bottom bar 0px, SOS `bottom` 20px. All four dismissal paths land offscreen: X, backdrop (`backdropClick.found true`), Escape, and a nav click (`/sos`). `contentClearsBar.ok` everywhere (`mainPad 80px` >= bar 62px) and `sosBottom 64px` > bar height. Public scene, all five themes: artwork request 200 / 129028 bytes, `background-repeat: repeat-y`, per-theme blur 12/18/22px, scrim opacity 0.88-0.96, artwork layer capped `height: 1800px`. Objective contrast (text hidden, rendered pixels behind each section heading): >= **12.06:1** at p99 across all five themes; artwork contribution to the brightest pixel behind section text <= +0.006 luminance on dark themes and -0.177 on daylight. **Not claimed:** real-device safe-area rendering — CDP `setDisplayFeaturesOverride` cannot emulate `env(safe-area-inset-*)`, so the inset plumbing was proven live only with `env()`=0 (header 12px, main 80px, bar 6px, SOS 64px). **Pre-existing bug found and fixed:** the minifier emitted `url('/images/nativity-carved-masks%20(2").webp')` (stray quote, 404) and `content:'"` on the `::after` scrim, also breaking `background-repeat`/`opacity` in built output; reproduced against a pristine `main` build, so it predates this change, and `origin/fix/auth-background-and-drawer-visibility:frontend/src/theme.css:304` still carries it. Fixed by renaming the asset (the minifier decodes `%28`/`%29` back to parens and drops the quotes, so percent-encoding does not hold). executed and verified before the credential existed; the branch is now pushed and open as **PR #67** (2026-10-18). **Merged** as `31eb19d` on 2026-10-18T11:36:41Z; PR #67 checks `frontend` and `race` both green. See `HANDOFF-2026-10-18-02`. |
| RES-005 | 2026-10-18 | `opencode` | **Recon** (`docs/community/GEO-SCOPED-FEED.md` §2, file/line evidence, no summary trusted): `ForumPost` (`models/Community.go:10-34`) had no geographic column; `GetForumPosts` (`community_handler.go:119-156`) filtered only nullable `unit_id`+role with **no limit**; `CreateForumReply` (`:198-263`) checked `IsLocked`+unit only. **Backend** (`backend/`, exit 0 each): `go build ./...`, `go vet ./...`, `go test ./...`; new `community_geo_test.go` **7/7 pass** (`TestScopeRankOrdering`, `TestPlaceVisibleToViewer`, `TestScopeWindowConditionWidensButNeverMoves`, `TestScopeWindowConditionForResolvedEmptyViewer`, `TestScopeSatisfiedBy`, `TestDefaultScopeFor`, `TestNewPlaceRecordsCoordinatesOnlyWhenGiven`, `TestPostVisibleTo`) — pins the ladder order, that same-LGA-different-state is NOT visible, that widening never emits another place, that the viewer place is parameterised not inlined, and that an unresolvable viewer sees only world. **Frontend** (`frontend/`, exit 0 each): `npm run typecheck`; `npm run lint` 0 errors / 3 pre-existing warnings; `npm test` 20 files / **185 passed**; `npm run build` OK. PR **#71**. **Not claimed:** live browser verification of the feed — this box has no Postgres, so no running backend to drive; UI behaviour is covered by the static gate and unit tests only. |
| RES-012 | 2026-10-18 | `opencode` | `callerLGAContext` now also returns the head-admin right (`membershipIsHeadAdmin`, the `IsHeadAdmin \|\| Role == UnitRoleAdmin` rule already used in `officers_handler.go`); `PostLGAChannel` returns **403 for non-head-admins**; new `lga_group_members` table with a partial unique index `(state, lga, unit_id) WHERE deleted_at IS NULL` created explicitly in `AutoMigrateAll` (same loud-failure posture as `idx_event_attendee_active`); three routes appended inside the existing `/community` block. `go build ./...`, `go vet ./...`, `go test ./...` exit 0. Frontend: `LGAChannelPanel` renders the composer only when the channel reports `canPost`, with join/leave + roster count from `GET /community/lga-group/members`; included in the same static gate above. PR **#71**. |
| RES-013 | 2026-10-18 | `opencode` | `docs/community/GEO-SCOPED-FEED.md` — grounded recon with file/line citations, the five-level scope ladder, the visibility predicate, the SQL window, the LGA-group design, the index/backfill plan, and an X × Facebook hybrid research section. **Research caveat recorded in the doc:** `websearch` returned no results in this environment and direct fetches returned only page chrome, so §6 is labelled *design reasoning from established product mechanics*, not cited sources — it is a hypothesis set to validate with users, not evidence. PR **#71**. **Research retry 2026-10-18T14:01Z (owner directive — second attempt):** `websearch` was retried with three distinct queries — _"Facebook audience selector vs X timeline hybrid feed design social product mechanics"_, _"hyperlocal community feed geographic scoping design"_, _"Nextdoor neighborhood feed design"_ — and returned `No search results found` for **every one**; a direct `webfetch` of `https://en.wikipedia.org/wiki/Nextdoor` returned only navigation chrome with no article body. §6 of the document is therefore left **exactly as it was**, still labelled design reasoning, and **no citations were fabricated**. |
| RES-014 | 2026-10-18 | `opencode` | **Recon (verified, not assumed):** `models.Report` is migrated (`config/migrate.go:98`) but **no Go code anywhere created one** — no handler, no route — so `POST /community/posts/:id/report` was a phantom endpoint and reports failed silently; the only caller was the **demo** `frontend/src/pages/CommunityPage.tsx:28` (the live page had no report control). `report_handler.go` is a different meaning of "report" (case/unit analytics). `GetForumPostByID` had `post.ViewCount++` + full-row `config.DB.Save(&post)` (`community_handler.go:244-245`) — read-modify-write lost update. RES-005 had already replaced the unbounded `Find` with a bounded `LIMIT`, so the residual paging defect was **offset** instability, not an absent limit. **Backend** (`backend/`, exit 0 each): `go build ./...`; `go vet ./...`; `go test ./...`. New `community_moderation_test.go` **6/6 pass** — `TestFeedCursorRoundTrip` (nanosecond + UTC fidelity), `TestFeedCursorIsURLSafe` (64 generated cursors contain no `+`/`/`/`=`, all round-trip), `TestDecodeFeedCursorRejectsGarbage` (empty / non-base64 / wrong field count / non-bool / non-timestamp / non-uuid / padded / standard-alphabet all rejected, plus trailing `=` rejected), `TestFeedCursorConditionMatchesFeedOrdering` (condition names `is_pinned`+`created_at`+`id`, exactly 3 forward `<` comparisons matching the feed's `DESC` order, placeholder count == arg count, cursor id parameterised not inlined), `TestIsCommunityModerator`, `TestCanModeratePost` (11 cases: super admin anywhere incl. other states; unit admin own LGA/state/own-unit-anywhere allowed; national / world / unknown-place / other-state / other-unit-national refused; citizen refused; nil user refused; unit admin with no resolved place refused). Also `TestFeedCursorIsURLSafe` guards the query-string contract. **Frontend** (`frontend/`, exit 0 each): `npm run typecheck`; `npm run lint` 0 errors / 3 pre-existing warnings (`useOfflineQueue.ts`, `ProfilePage.tsx`, `ReportIncidentPage.tsx`); `npm test` 20 files / **185 passed**; `npm run build` OK. Routes appended only inside the existing `/community` block; indexes `idx_report_active_target` (partial unique, `WHERE deleted_at IS NULL`) and `idx_reports_queue` created explicitly in `AutoMigrateAll` with the same loud-failure posture as `idx_event_attendee_active`. **CI proof:** PR #74 `race` job (Postgres 16) green, 6m27s — exercises `AutoMigrateAll`, so both new indexes were created against a real Postgres 16. **Merged** as `9d0bd8e` (PR #74, 2026-10-18). **Not claimed:** live browser verification — this box has no Postgres, so there is no running backend to drive; UI behaviour is covered by the static gate and the `race` job only. **Not in this PR (filed as RES-015):** content takedown/hide, ranking, scope-aware notifications, dead `ChatMessage` model, "communication room" membership authz scan, and the newly found unrouted demo `CommunityPage.tsx`. |

---

## Update protocol

1. **Claim:** set `Owner`, `Branch`, `Status = claimed`, `Started = now`.
   Announce in `agentmeeting.md` Section 6.
2. **Start coding:** push branch; set `Status = in_progress`; list real file
   paths in the board row.
3. **Open PR:** set `Status = in_review` and fill `PR`.
4. **Merge:** set `Status = done`, add an evidence row with test commands and
   results, move nothing — history stays in place.
5. **Block:** set `Status = blocked` and name the dependency (`after: RES-xxx`).
6. **Hand off:** write a `HANDOFF` entry, set owner to the receiving handle,
   status back to `claimed`.
7. **Every push that changes code must include the row update in the same push.**

---

## Completed (cycle archive)

| ID | Feature | Owner | Merged | Evidence |
|---|---|---|---|---|
| RES-001 | Global settings authorization | `claude` | PR #62 | See evidence log |
| RES-011 | Phone drawer width/insets + public background legibility | `opencode` | PR #67 (`31eb19d`) | See evidence log |
| RES-004 | RSVP concurrency / capacity safety | `opencode` | PR #69 (`9c1cf4f`) | See evidence log |
| RES-005 | Community geo-scoped feed (locality → LGA → state → country → world) + comment gating | `opencode` | PR #71 (`83f0493`) | See evidence log |
| RES-012 | LGA convergence group: join/leave/roster, head-admin-only posting | `opencode` | PR #71 (`83f0493`) | See evidence log |
| RES-013 | Community geo-feed research + architecture doc | `opencode` | PR #71 (`83f0493`) | See evidence log |
| RES-014 | Report + moderation queue, keyset pagination, atomic `view_count` | `opencode` | PR #74 (`9d0bd8e`) | See evidence log |
