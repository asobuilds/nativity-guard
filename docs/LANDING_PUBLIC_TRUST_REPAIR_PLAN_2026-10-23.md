# Nativity Guard: landing-page trust and functionality repair plan

**Audit date:** 2026-10-23  
**Scope:** public landing page, feature quick-view panels, public-data fallbacks, user-facing claims, and public policy navigation.  
**Principle:** When an API is empty or unavailable, display a truthful empty/error state rather than realistic demo records. Do not claim government certification, official dispatch, verified donations, legal compliance, or independent audit without evidence.  
**Change policy:** Implement on a feature branch, confirm CI and browser QA, then approve a reviewed deployment. No backend schema or dispatch behavior changes in this patch.

Status legend: **Patched – awaiting QA** means code was changed on this branch but has not passed full production browser testing; **Open** requires work or evidence; **Review** requires content or behavior validation.

| ID | Priority | Observed issue / verification gap | Status | Acceptance criteria |
| --- | --- | --- | --- | --- |
| NG-01 | P0 | Fictional patrol units shown as active and verified when API returns no units or fails | Patched – awaiting QA | API failure/empty response never displays sample units; labels depend on actual verification field |
| NG-02 | P0 | Unit donation modal substitutes a specific bank account on API failure | Patched – awaiting QA | On error, show no bank details and clear warning; do not enable donation for unverified units |
| NG-03 | P0 | Fictional supporter wall and donations shown as confirmed | Patched – awaiting QA | Only server records appear; no data/error states disclosed |
| NG-04 | P0 | SOS demo claims automatic dispatch, responder arrival, live GPS, benchmarks, and guarantees without independent evidence | Patched – awaiting QA | Guidance explicitly states non-emergency-service status; sign-in not confused with dispatch; call 112 advised |
| NG-05 | P1 | Nested donation dialogs may allow a single Escape press to close both child and parent; focus may leak | Patched – awaiting browser QA | Shared dialog stack, child portal, topmost Escape/Tab manager and unit tests; manual desktop/mobile checks remain |
| NG-06 | P1 | Published Terms/Privacy content is backend-dependent; production availability unverified | Patched – awaiting browser/API QA | Independent requests, active document validation, retry and section anchors; verify live rendered versions |
| NG-07 | P1 | About page does not publish complete legal operator/data-controller identification | Partially patched – identity evidence outstanding | Clearly disclose unverified identity; publish verified registered entity, controller and contact only after accountable review |
| NG-08 | P1 | Governance/immutable-audit/constitutional claims require backend behavior validation | Review | Substantiate every strong statement or mark as a design goal; no false guarantees |
| NG-09 | P2 | Homepage counts and geography can imply verified national coverage | Review | Meaning and source of each statistic are evident and current |
| NG-10 | P1 | Mobile drawer/popups not fully verified on Safari/Android, narrow screens and onscreen keyboards | Open | Test scroll, focus, all interactive controls, and accessible labels |
| NG-11 | P1 | Public map/search/error behavior can obscure API errors or show false data | Patched in part – awaiting QA | Distinguish unavailable/no-match; no invented map or case entries; end-to-end tests |
| NG-12 | P2 | Authentication popup action persistence, wrong credentials and cancellation need regression checks | Open | No stale pending action after failed login, close, or route change |
| NG-13 | P2 | Copyright/ownership information and rights/help contacts need publishing after verification | Open | Accurate owner/year, support and legal links, asset licensing reviewed |
| NG-14 | P2 | Popup/hash navigation and browser Back/Forward require browser QA | Open | Links, hashes, focus restore and Back/Forward pass on desktop/mobile |
| NG-15 | P1 | No verified basis for government endorsement/accreditation/guaranteed dispatch | Review | Show only evidenced claims; clearly state absence of endorsement and response guarantee |
| NG-16 | P0 | Fictional public incidents shown as current reports on API failure/empty response | Patched – awaiting QA | No made-up public reports in map or case tracking |
| NG-17 | P0 | Four fictional community warnings presented as live, verified advisories | Patched – awaiting QA | No fake local warnings in live panels; empty state until real authenticated feed wired |
| NG-18 | P1 | Public blueprint fallback embeds detailed security, encryption, and governance claims despite API failure | Open | Audit claims, remove unverifiable fallback, expose data failure transparently |
| NG-19 | P1 | AI safety sandbox contains simulation language implying validated officer credentials/services | Open | Label scenario examples; remove unverified badges and officer/response guarantees |
| NG-20 | P1 | Map can generate a default country-center location and describe it as captured coordinates | Patched – awaiting QA | Do not represent a default map center as visitor GPS; show location not chosen |
| NG-21 | P2 | Copy-to-clipboard UI can claim success even if clipboard permissions reject | Open | Success notification only after successful clipboard write; informative failure message |
| NG-22 | P1 | Urgent trust changes lack documented browser regression evidence | Open | Run frontend typecheck/lint/tests/build plus real-browser and mobile acceptance checklist |
| NG-23 | P1 | CI dependency install reports 6 npm advisories (3 moderate, 1 high, 2 critical) | Open | Review npm audit tree, direct vs transitive dependencies, reachable risk, patched versions, and regression testing; do not use --force blindly |
| NG-24 | P1 | Terms API returned a static acceptance version potentially differing from active Terms document | Patched – awaiting Go regression checks | API response version equals the served active document version; signup acceptance remains consistent |
| NG-25 | P0 | Existing Render Free Postgres expired on 2026-10-23, is suspended and makes backend fail startup | BLOCKED – recovery requires Render plan change | Upgrade existing database during grace period; preserve records; verify backend/API health before merges |

## Sequencing

### Phase 1 — public safety credibility (P0)
- Remove all fictional security units, incidents, donors, payment data, and warnings from production fallbacks.
- Replace unverifiable SOS marketing with clearly limited information.
- Show honest loading/empty/error states. Validate API and authenticated/donation behavior.
- **Done when:** every public view is truthful with a healthy API, empty API, and failed API.

### Phase 2 — interactive reliability (P1)
- Make nested modals keyboard-safe and test Escape, Tab, Shift+Tab, backdrop closing, scrolling and focus restore.
- Run public feature workflows, mobile viewport tests and case lookup failure states.
- **Done when:** desktop/mobile keyboard and click flows pass browser-based acceptance.

### Phase 3 — legal, governance and claims (P1/P2)
- Check Terms/Privacy backend route and document versions in production.
- Publish *verified* operator/controller details and contacts, confirm asset rights and copyright wording.
- Validate or remove unverified governance, encryption, AI advisor, financial and service claims.
- **Done when:** a reviewer can trace each public trust assertion to actual functionality or evidence.

### Phase 4 — sign-in readiness
- Only after Phases 1–3 are reviewed, continue sign-in UX refactor.
- **Done when:** CI, deployment health checks and real-device smoke tests pass.

## Phase 2 implementation notes

- The public quick panel and shared app Modal now register with a centralized dialog stack.
- Shared keyboard handling directs Escape and Tab only to the topmost open dialog.
- Child Modal renders in a body portal above the landing quick panel, preventing nested clipping.
- Background dialogs become inert and `aria-hidden` until the top dialog closes.
- Focus returns to the parent dialog's action trigger before the parent scroll lock is released.
- Focus management does not restart solely because an inline close callback changes identity.
- Mobile shared modal now scrolls within dynamic viewport height (`dvh`), with overscroll containment.
- New Vitest coverage checks order, Tab wrapping and a simulated nested Escape/focus/scroll sequence.
- Still required: real browser inspection on Chrome, Firefox, mobile Chrome and Safari; check repeated open/close, nested donate modal, Back/Forward and touch/keyboard behavior.

## Phase 3 implementation notes

- Legal pages load Terms and Privacy independently. A failing request cannot erase a healthy companion document.
- Documents are validated for kind/role, active status, content, version and effective date, with no synthetic legal text shown.
- `/terms#terms` and `/terms#privacy` are direct document anchors from the footer and About page; both sections expose retry actions.
- About page avoids guaranteed case/dispatch claims and accurately labels the in-app feedback link as sign-in-only.
- Published support email remains a *listed* contact, not an independently verified mailbox.
- The Go Terms handler now returns the current document's own version instead of a static version constant.
- Render app logs showed HTTP 200 for both citizen Terms and Privacy GET requests on October 20–21, 2026, but the exact payload and their availability on October 23 have **not** been independently verified. Full live browser/API QA remains outstanding.
- No legal identity, corporation registration, data controller, regulatory approval, or compliance status has been asserted without evidence.
- See `docs/LEGAL_PUBLICATION_VERIFICATION_CHECKLIST.md` for evidence/owner requirements.
- Previous Phase 1–2 PR #124 merged to main at 2026-10-23 09:15 UTC and Render frontend deployment is live. Phase 3 changes are on **separate draft PR #125**, not production.

## Production browser observation — 23 October 2026

- Real, read-only Chrome smoke testing of `https://nativityguard.com` confirmed the homepage, public Safety Map quick-view and About page loaded.
- The existing production `/terms` route rendered its heading but stayed on **"Loading…" for more than 20 seconds**. Neither policy request had settled during the test window; the cause is not yet established.
- **Release blocker:** cannot claim public Terms and Privacy contents are reliably available on production. This remains open until actual document content and versions load in a live browser.
- Phase 3 draft PR #125 now renders each policy independently, displays a clear warning after 8 seconds, aborts a still-pending request after 30 seconds and offers retry. It does not insert made-up legal text.
- These changes are draft-only until CI, review and deployment; they have **not** repaired the live site yet.
- Backend availability, frontend API origin settings, DNS and CORS should be checked with actual browser network evidence before claiming a root cause.

## P0 infrastructure blocker — database expired

As of 2026-10-23, Nativity Guard's existing Render Free Postgres instance is suspended after expiration (suspender: billing). Backend startup repeatedly fails while resolving the database's internal hostname. The live public Terms page was observed stuck on a loading indicator because both terms requests remained pending. This is not resolved by a frontend PR.

Render documentation allows a 14-day grace period to upgrade an expired Free database before deletion of its data. Recovery of the existing database and verification of backend health take priority over all other PR merges. No replacement database has been created.

See `docs/RENDER_POSTGRES_EXPIRATION_RECOVERY_2026-10-23.md` for data-preserving recovery steps. The current connected Render tool does not offer changing the database compute plan; a workspace operator must use Render's dashboard. Do not place database secrets in the issue tracker.

## Manual acceptance matrix

1. Empty /public/units: no patrol organisation or verified badge invented.
2. Failed /public/units: error message, no invented units.
3. Empty /public/cases: no sample incident or fabricated tracking ID in results.
4. Failed /public/cases: 'unavailable' instead of 'not found'.
5. Failed unit bank-accounts endpoint: zero account numbers shown and no 'Official Account' badge.
6. Missing platform donation-info endpoint: zero fallback account numbers shown.
7. Empty /public/platform/supporters: zero fake donors shown.
8. Failed supporter endpoint: understandable unavailable state.
9. Community Alerts panel: no fictional emergency/warning entries.
10. SOS panel: no assertion that signing in dispatches officers, no guaranteed response time.
11. All nested dialogs: Escape closes only topmost modal; focus remains inside active modal and returns to correct opener.
12. About, Terms, Privacy, links and controls: functional on desktop and mobile.
13. Hero verified-unit count: only real active and platform-verified API records used.
14. Report from map without location permission: never sends Nigeria-center coordinates as if captured.
15. Run npm audit and investigate transitive vulnerabilities in a separate reviewed dependency update.

## Verification status at authoring

GitHub source was inspected; Render showed a live frontend deployment on 2026-10-22. The live website and backend routes could not be accessed from this audit environment. Code changes on this branch do **not** prove the production deployment has been updated. CI, live API verification, external counsel for legal matters and actual browser testing remain outstanding.
