# Page performance audit and handoff

Reviewed main at 35b9ffa on 2026-10-01. This is a source and production-build audit, not a live latency benchmark. No claim that every page opens instantly is supported yet.

## Confirmed findings and changes

| Area | Evidence | Action |
|---|---|---|
| Navigation indicator | useRouteLoading starts a six-second timer on each pathname change; its minimum timer never clears loading | Removed the artificial banner from App; actual page/module and query loading states remain |
| Route code | Lazy pages sit under a global Suspense boundary | Added an inner boundary around AppShell content; header/sidebar remain available during module loading |
| Identity isolation | Shared QueryClient uses generic keys and survives logout/login | Clear cached queries at logout and successful login so one user cannot inherit another user's cached records |
| Session startup | AuthContext waits for /auth/profile before authorizing protected content | Preserve authorization; backend must measure profile and refresh latency |
| Case freshness | useCases and useCaseDetail inherit five-minute staleTime; no polling there | Backend/product owner must define freshness requirements; do not increase caching to disguise delays |
| SOS | List polls every 15 seconds; detail every 10 seconds | Preserve existing polling; validate actual dispatch freshness on staging |
| Maps | Separate Leaflet vendor chunk; geolocation, IP fallback and reverse geocoding add dependencies | Measure map UI, tiles, location and records separately; avoid blocking layout on permission |
| Unit governance | Policy/audit queries wait for membership-derived unit selection | Measure dependent calls; consider authorized summary endpoints rather than removing unit checks |
| Growing lists | News, alerts, announcements and unit handlers include unbounded Find calls | Backend should profile queries and add pagination with an agreed frontend contract |

## Production build observations

Representative gzip sizes from the local build: Leaflet vendor 45.16 kB, map page 2.96 kB, report page 9.05 kB, citizen home 3.11 kB, profile 5.23 kB. These are transfer sizes, not page opening times. Main entry and shared dependencies also contribute; do not add these numbers as a complete route budget without a network trace.

## Required live measurement matrix

Inventory all App.tsx routes (including aliases and disabled/placeholder pages). For each real page, use an authorized test account for its role and unit; test fresh browser cache, repeat navigation, mobile viewport and a throttled connection. Run each scenario at least five times. Record click-to-first-page-content, click-to-usable-data, request durations, transferred bytes, API errors and p50/p95. Separate full reload from in-app navigation. Capture browser traces and backend request IDs; never include tokens or personal case records in reports.

Priority journeys: citizen home/report/SOS/alerts/profile; officer queue/case/map; unit-admin cases/transfers/officers/policy; super-admin users/audit/analytics. Confirm page layout remains usable while data loads, and back navigation works during a slow chunk download. Check failure and retry states, denied location and offline behavior.

## Backend takeover

1. Review this branch without merging automatically. Confirm deployed frontend/backend commit and configuration; example env URLs do not prove the production target.
2. Measure profile, token refresh, case lists/details, unit memberships/policy, alerts and map data. Inspect database query plans and response size before selecting indexes or changing APIs.
3. Enforce role and unit authorization on every endpoint. Test two different users/units in the same browser with logout/login; cached records must disappear. Server authorization remains mandatory.
4. Agree pagination and freshness contracts, then coordinate frontend changes. Do not present cached emergency state as confirmation that a dispatch/action succeeded.
5. Run the measurement matrix and record before/after values. Existing unit tests and builds cannot substitute for live navigation tests.
6. Merge/deploy only after review and staging validation. Roll back this PR's commit if navigation regresses; preserve backend access controls.

## Verification status

Initial frontend build and 164 tests passed; lint had zero errors and three existing warnings. Final checks after the cache-isolation change are recorded in the PR. No authenticated live performance measurements or visual browser verification have been completed.
