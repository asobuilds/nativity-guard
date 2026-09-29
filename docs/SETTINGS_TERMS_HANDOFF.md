# Settings and user rules handoff — 2026-09-29

## Frontend available on this branch

- Every authenticated role can open `/settings` from the navigation. Accent (dawn, sky, forest) and text size save immediately in this browser's local storage and restore at startup. These settings do not imply account-wide sync. Emergency and lifecycle status colours remain unchanged.
- Existing profile editing, signed-in devices, notifications, citizen subscriptions, unit settings, and platform settings are linked to their existing routes. The page does not claim that unfinished controls exist.
- `/terms` is readable without signing in and has general rules plus citizen, officer, unit admin, and super admin guidance. Signup and login link to it. This is an **owner-review draft**, not a final binding agreement or a substitute for a privacy notice. No acceptance checkbox or inferred consent is recorded.

## Backend and product owner work for a binding release

1. Have the service owner and qualified Nigerian counsel review the wording, organisation identity, support/privacy contacts, limitations, dispute handling, enforcement/appeal process, effective date, version, and how material updates will be communicated. Publish a separate privacy notice accurately describing the actual data flows, purposes, retention, sharing, rights and contact process. The Nigeria Data Protection Act 2023 and NDPC GAID 2025 are relevant review sources: https://ndpc.gov.ng/ndp-act-2023/ and https://ndpc.gov.ng/wp-content/uploads/2025/07/NDP-ACT-GAID-2025-MARCH-20TH.pdf .
2. Provide a versioned public `GET /legal/terms` (or agree another contract) with effective date and role-specific text; preserve prior versions. Do not make the current static draft legally binding by adding a checkbox alone.
3. Add a server-side acceptance endpoint and record authenticated user ID, accepted version, timestamp, and source. On signup, decide how the created account accepts the version and ensure the server validates the version; do not trust a browser-only boolean. For existing users, show a review step when a materially new approved version applies. Allow the user to retrieve their acceptance history.
4. Define what happens when a person declines and provide a contact route. Privacy permissions that require separate consent should use separate choices, with server-side withdrawal handling; accepting general terms must not be treated as consent to every processing purpose.
5. If cross-device appearance preferences are wanted, add an authenticated preferences GET/PATCH scoped to the current user with strict allowed values. The current local preference is fully functional on the current browser; do not silently overwrite it when adding sync. Decide precedence and migration explicitly.

## Acceptance checks

- Test `/terms` anonymously and for each role; verify the correct guidance is visible and legible on mobile.
- Change accent and text size, reload, verify the choices restore; check keyboard navigation, legibility, case-status and SOS colours.
- Verify profile/device links against real authenticated endpoints. Test 401 and expired session handling.
- Before claiming binding acceptance, test version mismatch, concurrent updates, role changes, declined terms, and historical audit retrieval against the real backend. Record staging and production revisions/results.
