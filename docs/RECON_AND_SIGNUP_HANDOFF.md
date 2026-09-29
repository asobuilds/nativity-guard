# Nativity Guard recon and signup handoff — 2026-09-29

## Review scope

Reviewed `main` at `d4abb0b` and the frontend signup fix on this branch. This is a source review and local build/test verification; it is not evidence of a live deployment. Do not merge or enable unrelated features based on this document alone.

## Signup fix in this PR

- `POST /api/v1/auth/register` creates a citizen but returns no token. Previously `SignupPage` always redirected to `/auth/login` after a successful registration.
- The page now calls the existing `AuthContext.login(email, password)` after registration. That request uses `POST /auth/login`, stores the access and refresh tokens, fetches the profile, and establishes authenticated state. The page then navigates to `/`, the citizen home.
- If registration fails, the original server error is shown without an attempted login. If registration succeeds but login fails, the page routes to login with the email prefilled and the account-created notice. It does not retry registration or claim the account failed.
- Public signup still hardcodes the citizen role on the server; no privileged role can be chosen on this screen.

### Backend teammate: live acceptance checks

1. Review this PR and deploy frontend and backend to staging. Verify both `POST /auth/register` and `POST /auth/login` work from the frontend origin with a fresh test identity. Do not use a production person's credentials.
2. Sign up once, confirm the browser lands on the citizen home without a second password entry, then refresh and confirm the session restores via `/auth/profile`. Confirm the account appears once in the database and has the citizen role.
3. Check duplicate email/phone refusal, invalid form input, network loss on registration, and login failure *after* registration. The latter must show a login path and must not suggest registering again.
4. Inspect session/refresh-token creation, expiry, revocation, rate limits, and logs for this two-request flow. If registration and immediate login are blocked by the shared IP rate limit or other policy, coordinate a backend adjustment before release. Never return raw credentials or tokens in logs.
5. Repeat on the production frontend origin only after staging passes; record deployed revisions and actual results. The local build/tests do not prove live behavior.

## Other gaps found in this source review

| Area | Observed state | Next owner and release gate |
|---|---|---|
| AI assistant | `/assistant` is present but `VITE_ENABLE_AI_ASSISTANT` defaults off. Existing handoff: `docs/AI_FRONTEND_HANDOFF.md`. | Backend must address case authorization, data sharing, limits, provider configuration and live contracts. Frontend/backend jointly test before setting the build flag. Do not enable it in this PR. |
| Awareness | Alerts/news/subscriptions have API calls and mock-mode warnings; the project checklist still records live integration and push delivery as partial. | Backend/deployment owner confirms live endpoints, permission and push delivery. Frontend checks authenticated, empty, failure and mobile states against that service. |
| Officer and admin | The checklist records communications and parts of the staff consoles as open. The `on_scene → investigating` transition has no registered backend route, per `frontReadme.md`. | Backend defines and tests missing routes and role rules; frontend connects only to confirmed contracts. |
| Offline | Service worker and offline queue code exist; checklist still calls for field hardening, device sync and end-to-end tests. | Joint device testing must cover reconnect, duplicate writes and failed replay before claiming offline reliability. |

The first deliverable here is the signup fix. These remaining areas are separate releases with their own live acceptance checks, not features implicitly completed by this PR.
