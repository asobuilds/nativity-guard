# Media attachments on report and SOS — recon + implementation plan

> **Status:** recon + plan only. **No code was written by this document.** The plan is
> the deliverable; implementation waits for the owner to approve it.
>
> **Ledger:** `RES-017` (`recon/media-attachments-report-sos`, stacked on
> `docs/community-feature-audit` / PR #77).
> **Author:** `opencode`. **Source read at `9b13b3d`.**
>
> The owner stated voice/video already exists in some paths. **It does** — this document
> finds that path and proposes reusing it, rather than inventing a parallel one. Where no
> existing path exists, that is stated plainly.

---

## 0. Method and honesty notes

- Every claim carries `file:line`, read from the tree at `9b13b3d`.
- "Does not exist" claims are backed by a search for the construct that would exist if the
  claim were false (e.g. "no duration enforcement" is backed by searching
  `ffprobe|ffmpeg|duration` across `backend/`, not by failing to notice it).
- **No runtime verification.** This box has no Postgres and no `MEDIA_*` credentials, so
  no upload was performed against a live bucket. Storage-layer behaviour (presigned PUTs,
  `Head`, `FindByHash`) is read from code, not exercised.
- `websearch` was **not** attempted for this document — it has returned nothing twice this
  cycle (recorded in the `RES-013` evidence row), and nothing here depends on external
  citations.

---

## 1. Existing media infrastructure (the answer to "find it, don't invent it")

**Yes — there is a complete, generic media layer already, and it already names voice notes
and accepts video/audio MIME types.**

### 1.1 Storage backend

| Fact | Evidence |
|---|---|
| Service named `R2Service`; **talks to a Supabase Storage bucket over the S3-compatible API** | `services/r2_service.go:21-26` |
| Configured entirely from `MEDIA_*` env (`MEDIA_ENDPOINT`, `MEDIA_ACCESS_KEY_ID`, `MEDIA_SECRET_ACCESS_KEY`, `MEDIA_BUCKET`, `MEDIA_PUBLIC_BASE`, `MEDIA_REGION`) | `services/r2_service.go:60-66` |
| `R2Configured()` gate | `services/r2_service.go:100-104` |
| **Local-disk fallback** with `STORAGE_ROOT` (default `./storage`) when `MEDIA_*` is unset | `services/file_storage_service.go:31-36` |
| Key layout is identical in both modes: `<category>/<yyyy>/<mm>/<hash>.<ext>` | `services/file_storage_service.go:88-143` (`Save`/`storageKey`), `handlers/evidence_presign_handler.go:142` |
| **Content-addressed (SHA-256), deduplicating** | `services/file_storage_service.go:113`; `services/r2_service.go:268` (`FindByHash`) |

### 1.2 Key prefixes are already split public / private — and `voice_notes/` is already private

| Class | Prefixes | Evidence |
|---|---|---|
| public (CDN, `MEDIA_PUBLIC_BASE`) | `avatars/`, `covers/`, `units/`, `news/`, `community/` | `services/r2_service.go:37-43` |
| **private** (presigned GET only) | `evidence/`, `gov_ids/`, **`voice_notes/`** | `services/r2_service.go:46-50` |

### 1.3 Storage categories already include `voice_note`

`allowedCategoryDirs` (`services/file_storage_service.go:75-83`):

```
avatar → avatars      evidence → evidence    unit_logo  → unit_logos
cover  → covers       gov_id   → gov_ids     unit_cover → unit_covers
                      voice_note → voice_notes/
```

So `<category>` → `<dir>` is a simple registry, and **`voice_note` is already a member.**

### 1.4 Two upload paths already exist

**(a) Single-step multipart** — `POST /evidence/case/:caseId/file`
(`routes.go:210`), middleware `UploadValidationMiddleware("evidence")` + `CanAccessCase`,
handler `UploadEvidenceFile` (`handlers/evidence_upload_handler.go:51` `FormFile`,
`:70-71` `storageSvc.Save(file, "evidence")`).

**(b) Presign → client PUT → confirm** — the modern path, for large files
(`routes.go:211-212`):

| Step | Handler | Key behaviour |
|---|---|---|
| `POST /evidence/case/:caseId/presign` | `PresignEvidenceUpload` (`handlers/evidence_presign_handler.go:69`) | MIME allowlist `:106-109`; size ≤ `evidenceMaxBytes` = **100 MB** `:21,:115-118`; SHA-256 required `:120-124`; key `evidence/<yyyy>/<mm>/<sha><ext>` `:142`; PUT TTL **15 min** `:24` |
| client `PUT` | — | bytes go **straight to the bucket**, never through the API |
| `POST /evidence/case/:caseId/confirm` | `ConfirmEvidenceUpload` (`:170`) | re-`Head`s the object and verifies **actual size** `:220-231` and **actual type** `:234-239`; requires the key to be a private `evidence/` key `:204-207`; writes the `Evidence` row `:253-265` |

**The presign MIME allowlist already contains video and audio** (`evidence_presign_handler.go:32-45`):

```
video/mp4, video/webm, video/quicktime, audio/mpeg, audio/wav, audio/ogg, audio/webm
```

### 1.5 The read path already serves video/audio extensions privately

`GET /files/:category/:hash` (`routes.go:141`, `AuthMiddleware`) → `ServeFile`
(`handlers/file_serve_handler.go:41`).

| Rule | Evidence |
|---|---|
| public categories → 302 to CDN | `:74-82` |
| private categories → 302 to **presigned GET**, TTL **10 min** | `:86-94`, `:19` |
| `evidence/*`, **`voice_notes/*`** → require `caseId` query param + case access | `:121-152` |
| `gov_ids/*` → admin only | `:153-160` |
| `contentTypeFromExt` already maps `.mp4 .webm .mov .mp3 .wav .ogg .m4a` | `:166-196` (`.mp4` `:179`, `.mp3` `:186`, `.m4a` `:192`) |

### 1.6 …but nothing actually uploads a voice note today

Searching `backend/` and `frontend/src/` for `voice_note|voicenote` returns **only** the
storage registry (`file_storage_service.go:80`), the private-prefix list
(`r2_service.go:49`), and the serve-authorization comment/branch
(`file_serve_handler.go:30,35,121`). The only `storageSvc.Save(..., "<category>")` calls in
the codebase are:

| Call | Category | Evidence |
|---|---|---|
| `evidence_upload_handler.go:71` | `evidence` | |
| `user_media_handler.go:33` / `:81` | `avatar` / `cover` | |
| `unit_media_handler.go:50` / `:100` | `unit_logo` / `unit_cover` | |

**Conclusion:** `voice_notes/` is *wired but unused*. The voice-note upload path, model
column, route and UI do **not** exist. Video has no dedicated category at all — video is
accepted only under the generic `evidence/` prefix, whose serve-authz requires a `caseId`.

### 1.7 Frontend: no recorder exists, but the upload client does

- No `MediaRecorder`, `getUserMedia`, or `navigator.mediaDevices` anywhere in
  `frontend/src`. Every file input is `accept="image/*"`
  (`AvatarUpload.tsx:140`, `CoverUpload.tsx:138`, `UnitMediaUploader.tsx:100-101`,
  `citizen/ReportIncidentPage.tsx:650`). **There is no audio/video capture UI.**
- **But the presign → PUT → confirm client is already implemented** in
  `citizen/ReportIncidentPage.tsx:150-194` (hash at `:150`, presign `:165`, `PUT` `:172`,
  confirm `:188`). That is the client pattern to copy — it should be extracted, not
  rewritten.

---

## 2. SOS path

### 2.1 Today

| Fact | Evidence |
|---|---|
| `SOSAlert` model — **no attachment column**; free-text `Description` only | `models/SOSAlert.go:9-45`, `Description` `:17` |
| Create: `POST /sos/send` → `SendSOSAlert`, **JSON body** (`latitude`, `longitude`, `description`, `unitId`, `emergencyContacts`, `medicalInfo`, `priority`, `hideLocation`) | `routes.go:226`, `handlers/sos_handler.go:19-36` |
| No media/upload route in the `sos` group | `routes.go:226-239` |
| Response shape omits any media | `handlers/sos_tracking_handler.go:106-113` (`sosView`) |
| **Read predicate (reuse this)** | `sosReadScope` `handlers/sos_tracking_handler.go:74-87`; `sosRead` `:125-142` |

`sosReadScope` is the D-014-equivalent boundary for SOS: the alert owner, responders in the
caller's units, the alert's own unit's admins, super admins, and notified units for open
escalated alerts. **This is the predicate an SOS attachment must pass.**

### 2.2 Minimal change to add one voice note + one short video per SOS alert

1. Two nullable columns on `SOSAlert` (fixed slots, mirroring the avatar/cover precedent
   rather than a child table):
   - `VoiceNoteKey string` (object key under `sos_media/`)
   - `VideoKey string` (object key under `sos_media/`)
2. New private category `sos_media` → `sos_media/`.
3. Presign endpoint (caller-scoped, because the alert does not exist yet):
   `POST /sos/media/presign`, reusing `evidencePresignMIME` + a new, smaller size cap.
4. `POST /sos/send` gains optional `voiceNoteKey` / `videoKey`; `SendSOSAlert`
   **`Head`-verifies** each key (prefix, size, MIME) before persisting — the same
   verification `ConfirmEvidenceUpload` does at `evidence_presign_handler.go:220-239`.
5. Read: extend the `authorizeFileCategory` serve branch for `sos_media` to call
   `sosReadScope` (§5).

---

## 3. Report path

### 3.1 What "report" means here

RES-014's community report is `POST /community/posts/:id/report` → `ReportForumPost`
(`routes.go:349`, `handlers/community_moderation.go:90`). It writes the **shared generic**
`models.Report`:

| Fact | Evidence |
|---|---|
| `Report` model — `ReportedBy`, `TargetType` (`case|sos|announcement|user`), `TargetID`, `Reason`, `Status`, `ReviewedBy` | `models/Report.go:9-19` |
| **No attachment field** | `models/Report.go:9-19` |
| RES-014 sets `TargetType = reportTargetForumPost` (so `forum_post` is a fifth target) | `handlers/community_moderation.go:124-131` |
| Uploaded media would be readable only by someone who can see the parent post — **existing predicate**: `communityViewablePost` (`:45`) / `postVisibleTo` (`community_geo.go:149`), and moderators via `canModeratePost` (`:167`) | |

> Note: this is distinct from the **citizen incident report** (`ReportIncidentPage.tsx`),
> which is case creation and *already* uploads images as case `Evidence`
> (`ReportIncidentPage.tsx:150-194`). That flow is untouched by this plan.

### 3.2 Minimal change to add one voice note + one short video per report

1. Two nullable columns on `Report`: `VoiceNoteKey`, `VideoKey`.
2. New private category `report_media` → `report_media/`.
3. Presign endpoint gated by the **existing report visibility**
   (`communityViewablePost`), so a reporter can only presign for a post they can see.
4. `POST /community/posts/:id/report` gains optional `voiceNoteKey` / `videoKey`;
   `ReportForumPost` `Head`-verifies both before creating the `Report` row.
5. Read: extend the serve branch for `report_media` with a parent-report check (§5).

---

## 4. Storage constraints, and proposed limits

### 4.1 What is enforced today

| Constraint | Value | Evidence |
|---|---|---|
| Default / general max | 10 MB | `middleware/upload_validation_middleware.go:16` |
| Image max | 5 MB | `:17` |
| Document max | 10 MB | `:18` |
| **Evidence max** | **100 MB** ("video, audio") | `:79-81`; mirrored `evidence_presign_handler.go:21` |
| Image MIME allowlist | jpeg, png, webp, gif | `:21-26` |
| Document MIME allowlist | pdf, jpeg, png | `:28-32` |
| Evidence (single-step) MIME | **any detected type**; extension must match if detected | `:98-100`, `:111`, `:113-115` |
| Evidence (presign) MIME allowlist | images + video + audio + pdf | `evidence_presign_handler.go:32-45` |
| Presign PUT TTL / private GET TTL | 15 min / 10 min | `:24`; `file_serve_handler.go:19` |
| **Duration limit** | **none anywhere** | `ffprobe\|ffmpeg\|duration` search returns only request-log timing (`structured_logger_middleware.go`) |
| **Orphan / unconfirmed-upload cleanup** | **none** | `orphan\|unconfirmed\|cleanup.*(file\|upload\|media)` search returns nothing; the scheduler (`services/scheduler_service.go:95-307`) has no storage job |

Two consequences worth stating plainly:

- **The single-step `evidence` path performs no real MIME check on audio/video.**
  `magicSignatures` (`:34-40`) contains only image + PDF signatures, so an audio/video
  header sniffs as `""`; `:98` only rejects `""` when the category is *not* `evidence`;
  and `:113` skips the extension check when `detected == ""`. Video and audio therefore
  pass that path **unvalidated**. The **presign** path is the stricter one (it trusts the
  client's declared type but enforces the allowlist up front, then re-checks the stored
  type with `Head`). **This plan uses the presign path.**
- **Nothing garbage-collects un-attached uploads.** A presigned PUT that is never attached
  leaves an orphan object in the bucket forever.

### 4.2 Proposed limits, justified against the existing infrastructure

The owner suggested *voice ≤ 60 s / 5 MB* and *video ≤ 30 s / 20 MB*. Both are
**compatible with what is already here**, with one honest caveat: **duration cannot be
enforced server-side** — there is no probe (`ffprobe`/`ffmpeg`) in the stack, so duration
is a client-side constraint and a size cap is the server-side proxy.

| Media | Server-enforced cap | Why, against existing infra |
|---|---|---|
| Voice note | **5 MB** | Equals `MaxImageSize` (`:17`), so no new size tier is introduced. 60 s of Opus/AAC at 64–96 kbps ≈ 0.5–0.7 MB, so 5 MB is ~7–10× headroom and comfortably implies ≤ 60 s of speech. Reuse `audio/mpeg, audio/wav, audio/ogg, audio/webm` from `evidence_presign_handler.go:40-43`. |
| SOS/report video | **20 MB** | 1/5 of the existing `evidence` 100 MB cap (`:79-81`). Deliberately *below* evidence: this is a situational hint, not evidence-of-record, and 20 MB keeps the upload feasible on a poor mobile network during an emergency. 30 s at ~5 Mbps ≈ 19 MB, so 20 MB ≈ 30 s of typical phone video. Reuse `video/mp4, video/webm, video/quicktime` (`:37-39`). |

Rules for the plan:
- Enforce **size** server-side (declared size at presign `evidence_presign_handler.go:111-118`
  **and** actual size by `Head` on attach, mirroring `:220-231`).
- Enforce **MIME** server-side via the existing allowlist, on both declare and confirm.
- Treat **duration** as the client's responsibility; optionally embed a duration hint in
  the presign body and reject implausible values, but do not claim it is enforced.
- **Do not** delete media when a parent is deleted, until a reference count exists (see
  §5.3) — the store is content-addressed and shared.

---

## 5. Security

### 5.1 The rule

An attachment must not be readable by anyone who cannot read its parent. Concretely:

| Parent | Read predicate (reuse, do not re-implement) | Evidence |
|---|---|---|
| SOS alert | `sosReadScope(db, user)` | `handlers/sos_tracking_handler.go:74-87` |
| Community report | the report's parent post is visible (`postVisibleTo`) **or** the caller `canModeratePost` | `handlers/community_geo.go:149`; `handlers/community_moderation.go:45,167` |

### 5.2 Where the check lives

The platform's established read pattern is one generic route
`GET /files/:category/:hash` (`routes.go:141`) with a per-category branch in
`authorizeFileCategory` (`handlers/file_serve_handler.go:116`) that takes the parent id as
a **query param** — that is exactly how case evidence already works (`caseId`, `:122-152`).
Extending that function with two branches is the consistent choice:

- `sos_media` → require `?sosId=`; load via `sosReadScope(...).First(...)`; 404 if the
  caller cannot see the alert.
- `report_media` → require `?reportId=`; load the `Report`, then the parent post; allow if
  `postVisibleTo(user, post, nil, nil)` **or** `canModeratePost(user, post, place)`.

Both predicates already exist in package `handlers`, so this is reuse, not new logic.

Upload-side authorization:
- SOS presign must be behind `requireMinorApproved` (the guard already used by
  `SendSOSAlert` `handlers/sos_handler.go:20` and the evidence handlers).
- Report presign must require `communityViewablePost` success.
- Attach must reject any key that is not a private key under the correct prefix
  (mirroring `evidence_presign_handler.go:204-211`), which also blocks `..` and `//`.

### 5.3 A real caveat the store's dedup introduces

The store is **content-addressed with no reference count**
(`services/file_storage_service.go:113`; dedup lookup `services/r2_service.go:268`).
Therefore:

- Identical bytes under the same category **share one key** across different parents. Reads
  still require the caller to satisfy *a* parent check, so this is not a cross-parent leak:
  the bytes are identical, and the caller already legitimately holds them via the parent
  they can see. It is, however, worth documenting so nobody "fixes" it into a leak later.
- **Deleting a parent must not blindly delete the key**, or it will break every other
  parent that shares those bytes. The existing code already deletes old files on
  replacement (`user_media_handler.go:44,49`; `:92,97`), which is safe for single-owner
  avatars but is **not** safe here. The plan proposes **no automatic delete** for
  sos/report media until a refcount exists; flagged as a follow-up.

---

## 6. Implementation plan (no code written — this is the deliverable)

### 6.1 Backend

| # | Change | File | Detail |
|---|---|---|---|
| B1 | Add media columns to `SOSAlert` | `backend/models/SOSAlert.go` | `VoiceNoteKey string \`gorm:"type:varchar(255)" json:"voiceNoteKey,omitempty"\``; `VideoKey string \`gorm:"type:varchar(255)" json:"videoKey,omitempty"\`` |
| B2 | Add media columns to `Report` | `backend/models/Report.go` | same two columns |
| B3 | Register two private categories | `backend/services/file_storage_service.go:75-83` | `"sos_media": "sos_media"`, `"report_media": "report_media"` |
| B4 | Register two private prefixes | `backend/services/r2_service.go:46-50` | add `"sos_media/"`, `"report_media/"` |
| B5 | SOS presign | new `backend/handlers/sos_media_handler.go` | `PresignSOSMedia`; body `{filename,contentType,sizeBytes,sha256}`; reuse `evidencePresignMIME` + `requireMinorApproved`; caps: video 20 MB, audio 5 MB |
| B6 | SOS attach | `backend/handlers/sos_handler.go` (`SendSOSAlert`, `:19`) | accept optional `voiceNoteKey`/`videoKey`; `Head`-verify prefix+size+MIME; store on the row |
| B7 | Report presign | new `backend/handlers/community_media_handler.go` | `PresignReportMedia`; gate on `communityViewablePost`; key under `report_media/` |
| B8 | Report attach | `backend/handlers/community_moderation.go` (`ReportForumPost`, `:90`) | accept optional keys; `Head`-verify; store on the `Report` row |
| B9 | Serve authz | `backend/handlers/file_serve_handler.go` (`authorizeFileCategory`, `:116`) | new `sos_media` and `report_media` branches reusing `sosReadScope` / `postVisibleTo`+`canModeratePost`; add both to the doc comment `:30-38` |
| B10 | Surface keys in responses | `handlers/sos_tracking_handler.go:106-113` (`sosView`); `community_moderation.go` report view | add the two keys (so the UI can render) |
| B11 | Routes — SOS (append inside the existing `sos` block, `routes.go:239`) | `backend/routes/routes.go` | `sos.POST("/media/presign", Auth, requireMinorApproved-wrapped, handlers.PresignSOSMedia)` |
| B12 | Routes — community (append inside the existing `/community` block, `routes.go:351`) | `backend/routes/routes.go` | `community.POST("/posts/:id/report/presign", Auth, handlers.PresignReportMedia)` |
| B13 | Migration | `backend/config/migrate.go` | **no new model**, so `AutoMigrateAll` (`:9`) is unchanged; the two new columns ride on the existing `SOSAlert` / `Report` entries (`:49` / `:98`). Optional partial indexes are **not** needed (no new query paths keyed on media) |

Note on upload UX: SOS presign is `POST /sos/media/presign` (no alert id yet, because
`/sos/send` is one JSON call). The report presign is keyed by the *post*
(`/community/posts/:id/report/presign`) because the report row is created by the same
request that carries the keys. Both mirror `evidence_presign_handler.go`'s shape.

### 6.2 Frontend

| # | Change | File | Detail |
|---|---|---|---|
| F1 | Extract the upload client | new `frontend/src/hooks/useMediaUpload.ts` | lift the presign→`PUT`→(confirm/attach) logic already written in `ReportIncidentPage.tsx:150-194`; do not rewrite it |
| F2 | Recorder component | new `frontend/src/components/ui/MediaRecorderField.tsx` | `MediaRecorder` + `getUserMedia({audio})` / `({video})`; client-enforced 60 s / 30 s and 5 MB / 20 MB caps; MIME set from the shared allowlist |
| F3 | SOS form | `frontend/src/pages/citizen/ReportIncidentPage.tsx` (SOS branch) | optional voice + video; reuse existing image-upload styling (`:639-668`) |
| F4 | Community report dialog | `frontend/src/pages/LiveCommunityPage.tsx` (report action) | optional voice + video |

### 6.3 Tests (names to add)

Backend:
- `TestPresignSOSMedia_RejectsDisallowedMIME`
- `TestPresignSOSMedia_RejectsOversize`
- `TestSendSOSAlert_AttachesVerifiedMedia`
- `TestSendSOSAlert_RejectsKeyOutsidePrefix`
- `TestServeFile_SOSMedia_RequiresSOSVisibility` (owner sees; unrelated user 403/404)
- `TestPresignReportMedia_RequiresPostVisibility`
- `TestReportForumPost_AttachesVerifiedMedia`
- `TestServeFile_ReportMedia_RequiresPostVisibilityOrModerator`
- `TestFileStorage_SOSMediaAndReportMediaCategoriesAllowed`
- `TestSOSMedia_DedupSharedKey_NotDeletedOnParentDelete` (guards §5.3)

Frontend: extend the existing gate — `typecheck`, `lint`, and the unit suite.

### 6.4 Follow-ups this plan deliberately does not include

- **Orphan GC.** No cleanup exists for un-attached uploads (§4.1). A scheduler job that
  lists `sos_media/`, `report_media/` (and `evidence/`) and removes objects with no
  referencing row should be a **separate row**.
- **Server-side duration enforcement.** Requires a transcoder/prober; out of scope.
- **Reference counting** for deduplicated objects (§5.3).
- **Content-type sniffing for audio/video** on the single-step path (§4.1) — this plan
  avoids that path, but the weakness remains and should be filed.

---

## 7. Decisions the owner should make before implementation

1. **Fixed slots vs child table.** This plan uses two nullable columns per parent
   (one voice, one video), matching the avatar/cover precedent. A child table would allow
   N attachments and match `Evidence`, but is more work and changes the response shape.
2. **Serve via the generic `/files/:category/:hash?parentId=` route (recommended, matches
   case evidence) or via dedicated `GET /sos/:id/media/:kind` endpoints.** The generic
   route is consistent; the dedicated route keeps the check next to the predicate with no
   query param.
3. **Whether the 5 MB / 20 MB caps are right**, given there is no server-side duration
   enforcement.
4. **Whether moderators should see report media at all**, or only the post visibility
   holds (a moderator who cannot see the post cannot see its report media under this plan).

---

## 8. What this recon did **not** establish

- No live upload was performed (no `MEDIA_*` credentials, no Postgres on this box).
- Whether a production bucket currently contains any `voice_notes/` objects was not
  checked (and is not assumed).
- No UI verification: there is no running backend or recording device here, so the
  recorder component described in F2 is a plan, not a tested artefact.
- `docs/community/GEO-SCOPED-FEED.md` §6 citations remain unverified; nothing here depends
  on them.
