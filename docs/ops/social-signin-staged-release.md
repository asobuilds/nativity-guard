# Nativity Guard — staged citizen sign-in release checklist

## Why
Signup asks for too many details at first contact. Social entry can reduce friction but **must not grant emergency/reporting rights before age eligibility, consent and contact information are completed**.

## What this PR implements
- Two verified provider ID-token flows: Google via Identity Services, and Apple via Sign in with Apple JS (conditional on provider configuration).
- Fresh server-generated, one-use login nonce expiring after five minutes. Backend validates public RSA JWKS signatures, issuer, audience, expiry, nonce and verified email; only immutable provider `sub` can identify returning users. Email matches do not automatically link an existing password account.
- On a **first** social sign-in, the provider ID token must be verified and the user must confirm a date of birth that meets the existing minimum registration age **before any user record or session is created**. Returning provider subjects do not repeat the age prompt. Newly created citizen accounts then start with `account_access_level=limited` and no privileged role. Auth middleware rejects **every authenticated route other than profile, complete profile, logout and account deletion** until profile completion.
- Profile completion requires name, contact phone, and current terms/privacy consent; the validated birth date and minor status cannot be replaced through this screen. Consent and account promotion share a database transaction.
- Existing password login and staff permissions are unchanged.
- Google/Apple buttons are only displayed when the backend advertises verified configuration.

## Provider setup (no credentials committed)
Google: Create a Web OAuth client in Google Cloud, authorize the production origin `https://nativityguard.com`, and set `GOOGLE_CLIENT_ID` on the Go backend.

Apple: Requires an Apple Developer account, a Services ID for Sign in with Apple, a verified website domain and registered return URL. Set `APPLE_SERVICE_ID` and `APPLE_REDIRECT_URI` on the Go backend; test popup and redirect handling on actual Safari/iPhone. If not configured, no Apple button appears.

Do NOT put Apple private keys, Google client secrets, or storage credentials in frontend code. The server uses only public client IDs to validate ID tokens.

## Staging release gates
1. Check schema migration (user access level, login challenges, subject identities). Confirm older password users remain `full`.
2. Sign in to a new Google/Apple identity: the server first requires a valid age-eligible date of birth; then one `limited` citizen is created with no profile completion recorded. Attempt API calls directly (report, SOS, evidence, admin, identity upload) and assert HTTP 403 `profile_completion_required` before age/consent.
3. Invalid provider token, wrong audience, forged algorithm, wrong issuer, old nonce, re-used nonce and unverified email must all fail. Provider signing-key outage must fail closed.
4. Existing password account email must not be linked automatically. Provider subject must remain stable, even if its display name changes.
5. Complete profile with an eligible birth date, phone and explicit terms/privacy consent. Refresh the session and confirm citizen dashboard/report permissions only then, while normal minor safeguards remain enforced.
6. Check the provider age challenge, ineligible birth date (no account created), stale terms, duplicate phone and lost connection all keep the account limited with no partial consent.
7. Test Google on desktop and Android, Apple popup/redirect on real iPhone Safari; verify mobile safe areas, private browsing and denied third-party cookies.
8. Test refresh/logout/session revocation on both social and password sessions. Test account deletion for a limited account.
9. Independently run the `docs/ops/sos-case-media-staging-checklist.md` media and emergency checks in isolated staging with dispatch disabled. Never send genuine emergency alerts for QA.
10. Monitor login errors and successful sign-ins after deployment, without logging ID tokens, passwords, private media or personally identifying data.

## Next feature: one-time email link
A one-time sign-in link needs a **verified transactional mail domain**, an SMTP/API sender, short-lived single-use hashed tokens, user/session binding, abuse/rate limits, and an anti-enumeration response. This PR does not claim magic links are already enabled.

## Production status
The available Render connection does not include the Nativity Guard services. Source and CI checks alone cannot verify the live hosting environment, social-provider registrations, storage persistence, SOS dispatch or device compatibility. Do not enable social clients or deploy until staging checks complete.
