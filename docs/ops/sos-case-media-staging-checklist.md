# Nativity Guard — incident/SOS recording deployment checklist

The SOS and case-media repair in PR #114 adds same-API upload fallbacks and private playback. **Passing CI does not verify a live emergency workflow**. These checks require an authorized staging environment; never send a genuine emergency alert just to test the site.

## Required deployment configuration

- Backend environment: `MEDIA_ENDPOINT`, `MEDIA_ACCESS_KEY_ID`, `MEDIA_SECRET_ACCESS_KEY`, `MEDIA_BUCKET` and `MEDIA_REGION` must match the *private* media bucket.
- On Render's ephemeral/free web instance, set `MEDIA_REQUIRE_REMOTE=true`. This makes uploads fail explicitly if a durable remote backend is missing; it avoids claiming evidence was saved to disposable disk. This flag is now set in `render.yaml`; verify **actual service environment**, which may not automatically reflect blueprint changes.
- `ALLOWED_ORIGINS` must include the actual frontend origins, including `https://nativityguard.com` and any verified staging domain.
- Frontend `VITE_API_URL` must point to the deployed backend; this is compiled at build time, so rebuild after changing it. Keep `VITE_USE_MOCKS=false` in production.
- The browser's direct object-storage presigned PUT needs provider CORS allowing the actual frontend origin, `PUT`, `Content-Type`, and signed `x-amz-meta-*` headers. If the provider cannot support those headers, requests should use the authenticated backend fallback. **Do not expose storage credentials or private object keys in frontend code or logs.**

## Safe staging validation

1. Use a test account in an isolated staging project with emergency dispatch and external contacts disabled or mocked.
2. Upload a small voice file and MP4/MOV in **Report Incident**. Verify the case receipt appears first, attachment indicators reach **done**, and each media item can be played by its reporter and an authorized case reviewer.
3. Simulate an unavailable presigned PUT (without changing live provider credentials), and verify the authenticated backend fallback stores files and attaches them; test both local and selected-media paths. Deliberately reject a file to confirm the UI presents an error and offers a retry.
4. For SOS, validate the upload and playback components using a **mocked /sos/send response or isolated staging dispatcher**. Never use the production emergency dispatch channel for QA. Verify an alert receipt never implies responder acknowledgement.
5. Check iPhone Safari (audio/mp4/M4A), Android Chrome (audio/webm), desktop Chrome/Firefox, camera/microphone refusal, cancelled selection, offline transitions, file size cap, and expired authentication. Confirm the browser and storage paths display actionable errors, not just `Failed to Fetch`.
6. Test security: anonymous access must fail; a user who cannot view a case/SOS must not read its media; a user who can view one parent must not read a file belonging to another.
7. In staging, reboot/redeploy the backend, then repeat playback of previously attached files to prove persistent storage. Check object count, status codes, and error logs for 401/403/CORS/5xx errors. Never log private recording content.
8. Watch backend memory consumption: the authenticated fallback currently buffers file bytes in memory. Prefer direct-to-storage uploads for large video, cap concurrency, and load-test in staging before scaling production.

## Unverified live state

These steps are **not** proof that `nativityguard.com` is healthy. Without the actual hosting deployment logs, a staging environment, and authorized test credentials, the exact live cause of `Failed to Fetch` and successful end-to-end persistence cannot be confirmed.

If a real emergency is happening, use local emergency services rather than relying on an unverified in-app SOS delivery.
