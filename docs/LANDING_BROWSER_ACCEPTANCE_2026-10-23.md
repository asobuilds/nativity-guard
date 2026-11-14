# Landing browser acceptance — 23 October 2026

This checklist is for Nativity Guard's public landing and policy pages. It is **not** a substitute for live deployment verification.

## Automated Chromium smoke tests

Isolated GitHub Actions workflow: `.github/workflows/landing-browser-smoke.yml`, script `frontend/scripts/landing-browser-smoke.mjs`.

- Desktop landing: 3 primary entrypoints and a distinct emergency notice
- Safety map quick view: Escape closes one panel and removes hash
- All 8 public URL hash entrypoints: correctly titled panel opens and closes
- Nested unit donation modal: Escape closes the child only, returns to the parent, then unlocks page scrolling
- Public `/terms` route: the actual page consumes two independently stubbed API policy responses
- Public units API failure: no sample patrols should appear
- Mobile width 375px: navigation drawer opens and reaches the case tracking panel

The seven core smoke checks run against locally served source with **isolated API fixtures**; they never contact real users, security units or bank services. A **separate non-blocking, read-only** check visits the existing deployed homepage, opens/closes the public map panel, and reads the public About and Terms pages. That check does not send reports, trigger SOS, sign in or initiate payments. A failed live check is recorded separately and must not be presented as a passing production verification.

## Manual acceptance requirements (still mandatory)

1. On the actual served PR or approved deploy: desktop Chrome and Firefox, mobile Chrome and Safari.
2. Every quick-view panel: open, focus first control, Escape, Tab, Shift+Tab, tap close, tap backdrop and browser Back/Forward.
3. Nested donation dialog: verify stacking and one-Escape-one-dialog, scroll within the inner modal, return focus to opener.
4. On small mobile devices: check onscreen keyboard, safe area, portrait/landscape, no horizontal overflow and drawer accessibility.
5. Public map: no auto GPS permission request; clear approximate IP-geolocation label; no fictional unit/case markers.
6. Terms and Privacy: both backend documents load, version/effective date correct, privacy anchor reaches privacy section, failure and recovery states.
7. Backend failure/empty results: no fake cases, alerts, banking details, supporters, or verification badges.
8. Clipboard: real HTTPS permission grant and refusal; `Copied` appears only for successful write.
9. Check safety disclaimers, operator and controller identity against authoritative evidence; do not infer certification from UI text.
10. Review browser console, network failures, layout stability and keyboard focus return.

## Merge readiness

- Draft smoke-test PR creates an automated first line of coverage but does not certify the live domain.
- PRs #125 (legal), #126 (governance/AI), #127 (hero/clipboard), #128 (dependency audit) require their own review.
- A transient integration deadlock or missing table must be understood; rerun alone is not conclusive root-cause analysis.
- Keep the landing flow review distinct from the sign-in redesign.

## Recording real-device results

For each environment, record date, device, OS/browser version, release SHA, pass/fail, issue ID and reproduction instructions. Link supporting screen recordings internally only if they contain no private incident, contact or payment details.
