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
| NG-05 | P1 | Nested donation dialogs may allow a single Escape press to close both child and parent; focus may leak | Open | Automated keyboard test + manual nested-modal check on desktop and mobile |
| NG-06 | P1 | Published Terms/Privacy content is backend-dependent; production availability unverified | Open | Both /terms documents load when API works; transparent failure state; published versions confirmed |
| NG-07 | P1 | About page does not publish complete legal operator/data-controller identification | Open | Publish verified entity, contact and responsible controller details after legal review |
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
| NG-20 | P1 | Map can generate a default country-center location and describe it as captured coordinates | Open | Do not represent a default map center as visitor GPS; show location not chosen |
| NG-21 | P2 | Copy-to-clipboard UI can claim success even if clipboard permissions reject | Open | Success notification only after successful clipboard write; informative failure message |
| NG-22 | P1 | Urgent trust changes lack documented browser regression evidence | Open | Run frontend typecheck/lint/tests/build plus real-browser and mobile acceptance checklist |

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

## Verification status at authoring

GitHub source was inspected; Render showed a live frontend deployment on 2026-10-22. The live website and backend routes could not be accessed from this audit environment. Code changes on this branch do **not** prove the production deployment has been updated. CI, live API verification, external counsel for legal matters and actual browser testing remain outstanding.
