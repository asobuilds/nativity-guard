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
| RES-004 | Community | RSVP concurrency / capacity safety | `opencode` | `fix/community-rsvp-concurrency` | `backend/handlers/community_handler.go` (RSVP path), `backend/models/Community.go`, `backend/config/migrate.go`, `backend/internal/testutil/setup.go`, new `backend/handlers/rsvp_concurrency_test.go` | `in_progress` | 2026-10-17T16:12Z | — |
| RES-005 | Community | Real Community moderation | `unclaimed` | `feat/community-moderation` | `backend/handlers`, `backend/models`, `frontend/src/pages/LiveCommunityPage.tsx` | `unclaimed` | — | — |
| RES-006 | Finance | Finance/bank route authorization, atomic approvals, API contract alignment | `antigravity` | `fix/finance-route-review` | `backend/handlers/{finance,bank_account,ledger,public_bank,platform_donation}_handler.go`, `backend/routes/routes.go` (aliases only), new `backend/handlers/finance_auth_test.go` | `claimed` | 2026-10-17T16:19Z | — |
| RES-007 | Docs | Reconcile stale hardening queue in README/AGENT.md | `unclaimed` | `docs/reconcile-hardening-queue` | `README.md`, `AGENT.md` | `unclaimed` | — | — |
| RES-008 | Security | Event RSVP object-level authorization (cross-unit + non-upcoming) | `opencode` | `fix/community-rsvp-authorization` | `backend/handlers/community_handler.go` (`RSVPToEvent`) | `blocked` (`after: RES-004`) | — | — |
| RES-009 | Frontend | Perceived reliability: global error surfacing, loading states, mutation feedback | `claude` | `feat/frontend-reliability-pass` | `frontend/src/lib/queryClient.ts`, `frontend/src/components/ui/States.tsx`, data-heavy pages | `claimed` | 2026-10-17T16:27Z | — |
| RES-010 | Infra | Backend build/vet CI gate (none exists today) | `unclaimed` | `ci/backend-build-vet` | `.github/workflows/` | `unclaimed` | — | — |

**Suggested split for the three agents** (a suggestion is not a claim — confirm
in `agentmeeting.md` before starting):

- `claude` — RES-002, then RES-003 (backend authorization, needs the broadest
  repo context).
- `antigravity` — RES-005 frontend half only once a backend contract exists;
  until then RES-006 review (read-mostly, no hot files).
- `opencode` — RES-004 (self-contained handler + migration), then RES-007.

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
| RES-004 | 2026-10-18 | `opencode` | Local: `go build ./...`, `go vet ./...`, `go test ./...`, `go vet -tags integration ./...` all pass. `TestRSVP` (`rsvp_concurrency_test.go`) executes in CI `integration-race.yml` against `postgres:16`; this box has no Postgres server. Push/PR not run here: no GitHub credentials (gh logged out; no token/SSH). See `HANDOFF-2026-10-18-01`. |

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
