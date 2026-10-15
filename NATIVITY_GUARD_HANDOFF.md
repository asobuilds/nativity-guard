# Nativity Guard — Project Handoff

Last updated: 2026-10-13

## Project
- Repository: https://github.com/asobuilds/nativity-guard.git
- Local path: C:\Users\gentle\nativity-guard
- Feature branch: feature/one-time-signin-reset
- Base branch: main

## Current objective
Implement Option C: separate one-time sign-in email links and password-reset email links in the existing Nativity Guard application.

The feature is NOT implemented yet. Continue from the actual source code. Do not assume an updated ZIP exists.

## Required features

### 1. One-time sign-in email link
- User requests a sign-in link by email.
- Generate a cryptographically secure, unpredictable token.
- Store only the token hash in the database.
- Enforce expiry and single-use consumption.
- Reuse the existing authentication/session creation flow.
- Reject invalid, expired, used, or deactivated-account tokens.
- Do not weaken existing authentication or authorization.

### 2. Password-reset email link
- Provide a separate password-reset email link.
- Use a different token purpose and storage/validation path from sign-in links.
- Store only a hash of the token.
- Accept the token from the frontend URL.
- Enforce expiry and atomic single-use consumption.
- Change the password and revoke existing sessions and refresh tokens.
- Avoid revealing whether an email belongs to an account.
- Preserve existing SMS/code recovery where practical.

### 3. Frontend
- Add separate actions for:
  - Email me a sign-in link.
  - Email me a password-reset link.
- Allow the reset page to read a token from the URL.
- Preserve existing login, signup, GIS, live location, SOS, incidents, and other functionality.

## Important source files

### Backend
- backend/handlers/auth_handler.go
- backend/handlers/email_handler.go
- backend/routes/routes.go
- backend/services/auth_service.go
- backend/services/email_service.go
- backend/services/password_reset_service.go
- backend/services/session_service.go
- backend/services/token_service.go
- backend/models/
- backend/main.go

### Frontend
- frontend/src/pages/auth/ForgotPasswordPage.tsx
- frontend/src/pages/auth/LoginPage.tsx
- frontend/src/pages/auth/ResetPasswordPage.tsx
- frontend/src/pages/auth/SignupPage.tsx
- frontend/src/lib/passwordReset.test.ts
- frontend/src/lib/apiClient.ts
- Auth context and router files

Inspect actual source files before editing.

## Existing reset behavior

`backend/services/password_reset_service.go` contains:
- `PasswordResetService`
- `RequestReset(identifier string)`
- `ResetWithToken(rawToken, newPassword string)`
- A `ResetNotifier` interface for email and SMS.
- A one-hour reset-token TTL.
- Session and refresh-token revocation after password reset.

The existing `RequestReset` accepts email or phone and silently ignores unknown accounts to reduce account enumeration. It stores a SHA-256 hash of the reset code and sends notifications through configured channels.

### Critical security issue to investigate
The existing `generateResetCode()` uses a one-byte random buffer to generate a six-digit code. This provides inadequate entropy and does not generate a proper six-digit distribution. Inspect the tests and fix this carefully.

For new email links, generate a separate high-entropy URL-safe token using `crypto/rand` (for example, 32 random bytes encoded with URL-safe base64). Store only the hash, never the raw token.

The existing reset model should be inspected before deciding whether to extend it or create a separate model for one-time sign-in tokens. Keep the two token purposes separate.

## Existing frontend behavior

`ForgotPasswordPage.tsx` currently calls:

`POST /auth/forgot-password`

with an `identifier` (email or phone), then navigates to `/auth/reset-password`.

`ResetPasswordPage.tsx` currently calls:

`POST /auth/reset-password`

with `token` and `newPassword`. It expects a six-digit code and logs out locally after successful reset because the backend revokes sessions.

The requested feature is email links, so adapt the reset page to support a token query parameter while considering compatibility with the existing code flow.

## Previously confirmed checks

- `go test ./...` passed in the backend.
- `npm.cmd install` succeeded in the frontend.
- `npm.cmd run build` succeeded in the frontend.
- PowerShell blocked `npm` due to execution policy; `npm.cmd` worked.
- Frontend build warnings included an unresolved image path, a CSS minification warning, and overlapping static/dynamic imports. The build still passed.
- npm reported 8 audit vulnerabilities: 3 moderate, 3 high, 2 critical. Avoid broad automatic upgrades without reviewing changes.

These are previous results, not proof of the current machine's state.

## First commands on the next machine

```powershell
cd C:\Users\gentle\nativity-guard
git status
git branch --show-current
git branch --all

f the feature branch exists, switch to it. If not, check the remote and current work before creating a new branch. Never discard existing changes without reviewing them.
Recommended implementation workflow
1. Inspect auth handlers, routes, email service, session/token services, and models.
2. Find exactly how regular login creates access/refresh tokens and sessions.
3. Implement separate endpoints and token handling for sign-in links and password-reset links.
4. Ensure tokens expire, are stored as hashes, and are consumed atomically.
5. Implement frontend forms and token-link handling.
6. Add tests for invalid, expired, reused tokens and deactivated accounts.
7. Run backend tests and frontend build/tests.
8. Review the diff and commit only after verification. Do not merge into main prematurely.
Working preferences
- Keep instructions practical and concise.
- Give copy-paste-ready commands.
- Move quickly and avoid unnecessary Git housekeeping.
- Build on existing working code.
- Preserve existing GIS/map work, live location, SOS, incidents, and authentication.
- Never claim code, tests, or ZIP files are ready unless actually verified.
Prompt for the next ChatGPT session
Continue my Nativity Guard project using this handoff. Implement separate one-time sign-in email links and password-reset email links in the existing Go backend and React/TypeScript frontend. First inspect the current source and repository state. Use cryptographically strong single-use tokens, store only token hashes, enforce expiry and atomic token consumption, reuse the existing session creation flow for sign-in, and revoke sessions after password reset. Preserve SMS/code recovery where practical and do not break GIS, live location, SOS, incidents, or existing authentication. Give me practical PowerShell commands and copy-paste-ready changes. Run backend tests, frontend build/tests, and review the diff before suggesting a commit. Do not assume an old ZIP exists.