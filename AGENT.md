# AGENT.md — Nativity Guard

> Operating contract for any AI or human contributor working in this repository.

---

## Who you are

You are an implementation agent for the **Nativity Guard** repository.

You are **NOT** the product owner. You are **NOT** authorized to redesign the architecture. You inspect, plan, implement, test, and report — within the existing architecture.

---

## Source of truth (highest priority first)

1. Existing working code
2. This file (`AGENT.md`)
3. `README.md`
4. Database schema and migrations
5. Existing API contracts
6. Explicit instructions from the user
7. Approved implementation plans

**Rule:** Never assume an apparently reasonable change is architecturally correct. Inspect the repository first.

---

## Security principles

### Core rule
**Unit access is NOT case access.**

An officer in a unit does **not** automatically gain access to every case in that unit. Sensitive case intelligence is available to:
- Assigned officers (`primary` / `paired`)
- Explicitly authorized admins (`CaseAdminAssignment`)
- Head Admin of the unit
- Super Admin

Support officers see a **limited** view of their assigned case only.

### Additional rules
- Sensitive suspect tracking is never a generic officer capability.
- Citizen-facing responses never expose officer identity, evidence internals, admin comments, or other suspects.
- Presumed innocence: a citizen named as a suspect sees only case ID, category, status, and public-safe progress.

---

## Role hierarchy

    Super Admin  (platform-level, outside unit hierarchy)
        ?
        ??? Unit
              ??? Head Admin  (1 per unit, elected by admins)
              ??? Admin       (5–10 per unit, elected by verified members)
              ??? Officer     (tier: primary / paired / support)
              ??? Member      (verified or provisional)

Pre-membership citizens are outside the unit hierarchy.

---

## Case lifecycle (canonical)

    pending ? assigned ? dispatched ? on_scene ? investigating
       ? pending_admin_review
       ? [admin_changes_requested ? investigating]
       ? closed

**Enforced rules:**
- Officers **cannot** close a case directly. Closure requires admin approval.
- Assigned officers **cannot** approve their own case closure.
- Closure requires **2 approvals** from the 2–3 submitted admins.

---

## Governance

### Elections
- Term: 12 months
- Staggered rotation: half the seats renew every 6 months
- Term limit: 2 consecutive terms, then 6-month cooling-off
- Quorum: 50% of eligible verified voters
- Seat bands: 12–20 ? 5 · 21–40 ? 7 · 41–70 ? 9 · 71–100 ? 10
- Vacancy fill: next-highest vote-getter from the last election

### Revocation
- Regular admin removal: **10+ verified votes + Head Admin approval**
- Head Admin removal: **majority of verified unit members**
- One vote per member per cycle
- Votes are immutable; tally is derived, never trusted from the client
- Successful removal triggers cooling-off

### UnitAuth
- Per-unit policy document: thresholds, quorum, seat bands, term rules, cooling-off
- Every election and revocation cycle snapshots the policy version at open time
- No retroactive rule changes

---

## Location model

- Map is a **central platform capability**, not a role-specific feature
- Public map: public-safe geographic information only
- Operational map: restricted by role and case assignment
- Sensitive case location: restricted to authorized case personnel

---

## Living Local Intelligence

    user location ? local context ? short question
        ? community observation ? validation ? knowledge fact
        ? retrieval ? AI reasoning

**Rule:** Nativity Guard learns **facts** before it learns **models**.

Never treat an AI inference as equivalent to a verified fact. All knowledge carries provenance, confidence, source type, and timestamps.

---

## Working rules

1. Inspect before modifying.
2. Never modify unrelated files.
3. Never overwrite architecture without approval.
4. Never invent APIs.
5. Never invent database fields when existing fields can be reused.
6. Never expose secrets.
7. Never commit secrets.
8. Never bypass authorization.
9. Run relevant tests after implementation.
10. Report every modified and created file.
11. Report tests executed and results.
12. Stop after completing the assigned task.
13. Do not automatically begin the next phase.
14. Do not perform destructive Git operations.
15. Do not commit unless explicitly instructed.

---

## Execution cycle

    READ ? UNDERSTAND ? PLAN ? REPORT
        ? WAIT FOR APPROVAL
        ? IMPLEMENT ? TEST ? REPORT ? STOP

---

## Before every task, provide

- OBJECTIVE (one sentence)
- FILES TO INSPECT
- FILES TO MODIFY
- FILES TO CREATE
- FILES NOT TO TOUCH
- IMPLEMENTATION PLAN
- SECURITY CONSIDERATIONS
- TEST PLAN
- RISKS

## After every task, provide

- FILES CHANGED
- FILES CREATED
- WHAT CHANGED
- TESTS RUN
- TEST RESULTS
- REMAINING RISKS
- NEXT POSSIBLE STEP

---

## Testing & Git

- Backend: `go build ./...`, `go vet ./...`, `go test ./...` must all pass
- Frontend: `npm run build` must pass
- No commit without explicit instruction
- No destructive Git ops (no force-push, no rebase, no reset --hard, no drop)

---

## Current production state

- Backend and frontend are deployed on Render; PostgreSQL is hosted on Supabase.
- Production domain: `nativityguard.com`.
- Community reads and writes are live. Treat `VITE_COMMUNITY_WRITES_ENABLED` as a UI/build flag only; backend authorization and validation are mandatory.
- Super-admin navigation must expose only real, usable capabilities. Do not reintroduce fake/Coming Soon fallbacks for implemented admin routes.
- Unit analytics authorization is scoped: super admins may inspect any unit; unit admins may inspect only their own unit.
- Super-admin user mutations already block self-role-change/self-suspension and validate role values.
- Next security work: global settings authorization, last-super-admin protection, RSVP concurrency/capacity safety, real Community moderation, finance-route review, and final role-by-role authorization audit.
- Every production-facing control must be wired to a real backend action or removed/clearly unavailable. Never present mock/demo behavior as live.

---

## Current phase rules

- One Kilo task per file (or two independent files max)
- Build + vet after every task
- Push every 5–10 tasks
- Docs update in the same commit as the change they describe
- Never let README.md or AGENT.md drift from reality

--- END AGENT.md ---

### Navigation regression guard (October 2026)

Preserve the existing carved-mask public background and all five user-selectable themes unless explicitly requested otherwise. In `frontend/src/components/layout/AppShell.tsx`, keep the mobile drawer, its close (X) button, backdrop, menu trigger, bottom navigation, and desktop docked sidebar aligned to the same `lg` breakpoint. Test close by X, backdrop, Escape and navigation, including medium-width tablet layouts and long super-admin menus. Do not describe UI as browser-verified solely because frontend CI passes. PR #63 is the regression repair; validate deployment status before claiming it is live.
