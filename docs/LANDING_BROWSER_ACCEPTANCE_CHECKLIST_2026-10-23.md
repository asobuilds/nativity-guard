# Nativity Guard — Landing browser release checklist

**Created:** 2026-10-23  
**Status:** Open release gate. Automated Chromium results are necessary, not sufficient.  
**PR:** #129 (browser acceptance tests); follows #124 (merged), and draft #125–#128.

## Automated baseline

GitHub Actions `landing-browser-qa` builds the application, serves the built frontend locally, and runs desktop/mobile Chromium Playwright smoke tests. All public API calls are intercepted with synthetic test fixtures. No live incident submission, payment, identity verification, or real unit is created by this suite.

Tests cover:
- Hero, three primary actions, SOS limitations and trust footer
- All eight feature hashes and Escape close behavior
- No fictitious units/incident records with empty public API responses
- Nested bank-details modal: child-only Escape, correct active dialog, focus and body scroll restoration
- No bank information on failed account endpoint
- Anonymous public About and Terms route rendering
- Mobile menu navigation to a public feature and safe closing

## Manual acceptance before merging feature PRs

### Desktop

- [ ] Chrome, Firefox and Safari (current supported releases): Open each feature via navbar, button and direct URL hash.
- [ ] Test `Tab`, `Shift+Tab`, `Escape`, backdrop click and focus restore.
- [ ] Open a test verified-unit donation dialog nested inside the quick-view; only the child closes on first Escape.
- [ ] Check in-app navigation plus browser Back/Forward without losing or re-opening wrong dialogs.
- [ ] Check appearance theme/wallpaper buttons, contrast and persistence.
- [ ] Check that API-down states remain visibly different from zero published records.
- [ ] Check the public map does not label fallback map center as visitor GPS.

### Mobile

- [ ] Real Android Chrome and iOS Safari: open/close drawer, search, map, case tracking and donation details.
- [ ] Verify no trapped body scroll, invisible controls, text overflowing viewport, or keyboard covering close/submit buttons.
- [ ] Check legal-page anchors, policy loading failures and visible contact details.
- [ ] Check accessibility names, screen reader announcements and no focus behind an active dialog.

### Public authenticity and safety

- [ ] Actual Terms & Privacy routes return published reviewed documents from live backend.
- [ ] No simulated patrols, incidents, supporters, emergency warnings or fictional bank accounts appear as real.
- [ ] SOS buttons are clearly distinguished from Nigerian emergency hotline 112 and do not guarantee dispatch.
- [ ] Financial/account copy action shows success only if clipboard API actually succeeds (PR #127).
- [ ] Governance/AI samples clearly marked illustrative, not verified operational promises (PR #126).
- [ ] Operator identity, rights, support inbox, controller contacts and third-party image rights verified before making legal compliance claims (PR #125).
- [ ] Assess known dependency advisories and dev-server exposure before closing NG-23 (PR #128).

## Release decision

PR #124 is already live; PRs #125–#128 should remain draft until applicable CI, browser QA and accountable reviews pass. QA fixtures verify only frontend behavior, **not** real backend availability, legal compliance, geographic coverage, or emergency dispatch.

Document the tested browser/OS, reviewer, test date and failed steps when sign-off occurs; do not mark manual items complete without evidence.
