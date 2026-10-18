# Community feature audit — Discussions, Announcements, Events

> **Status:** recon only. **No code was changed by this document.** Findings were read
> from the source at commit `9b13b3d` (`origin/main`, 2026-10-18), not inferred from
> earlier summaries (RES-005 / RES-013 / RES-014 prose was treated as a claim to
> re-check).
>
> **Ledger:** `RES-016` (`docs/community-feature-audit`).
> **Author:** `opencode`.

---

## 0. Method, and what "verified" means here

Every claim below carries `file:line`. Three techniques were used, in this order:

1. **Read the model, handler and route definitions directly** (`backend/models/Community.go`,
   `backend/handlers/community_handler.go`, `backend/routes/routes.go`,
   `backend/config/migrate.go`).
2. **Cross-check every frontend API call site against the registered routes
   mechanically**, rather than by eye. The checker parses `routes.go` tracking group
   prefixes, strips the `/api/v1` mount the frontend prepends
   (`frontend/src/lib/apiClient.ts:17-19`), and scans every
   `api.<method>(...)` call to the matching close paren so that string concatenation
   and template interpolation resolve correctly. Result: **299 backend routes,
   172 frontend call sites, 0 genuinely unmatched** — see §4.
3. **Verify the negative claims** (things that *don't* exist) by searching for the
   thing that would exist if the claim were false — e.g. "no event cancellation" is
   backed by searching for `CancelEvent|DeleteCommunityEvent|UpdateCommunityEvent`
   across `backend/`, not by failing to notice it.

**Not verified, and not claimed:** no runtime behaviour. This box has no Postgres, so
nothing here is a live request/response observation. Everything is static reading of
the code plus mechanical cross-checking. Where a claim depends on runtime state
(e.g. what rows currently exist in `community_announcements`), that is stated as
unknown rather than guessed.

---

## 1. Announcements

### 1.1 Is an announcement a pinned `ForumPost`?

**No.** It is a dedicated model with its own table.

| Fact | Evidence |
|---|---|
| `CommunityAnnouncement` struct | `backend/models/Community.go:79-101` |
| Table name `community_announcements` | `backend/models/Community.go:97-99` |
| Migrated | `backend/config/migrate.go:60` |
| Fields | `ID`, `UnitID`, `Title`, `Content`, `Type`, `Priority`, `AuthorID`, `Status`, `ExpiresAt`, `PublishedAt`, `CreatedAt`, `UpdatedAt`, `DeletedAt` |

`ForumPost` (`backend/models/Community.go:31-52`) is a **different** model with
`IsPinned`, `IsLocked`, `ViewCount`, `ReplyCount` and an embedded `GeoPlace`. The two
share no table and no code path. An announcement therefore **cannot** be a pinned
forum post, and a pinned forum post is not an announcement.

### 1.2 A second, dead announcement model exists

There are **two** announcement models:

| Model | Table | Migrated | Routed? |
|---|---|---|---|
| `CommunityAnnouncement` | `community_announcements` | `migrate.go:60` | Yes — `routes.go:333-334` |
| `Announcement` | `announcements` | `migrate.go:45` | **No** |

`backend/handlers/announcement_handler.go` defines `CreateAnnouncement` (`:13`),
`GetAllAnnouncements` (`:57`), `GetAnnouncementByID` (`:69`), `UpdateAnnouncement`
(`:88`) and `DeleteAnnouncement` (`:134`) — and **not one of them is registered on any
route.** Search across `backend/` for
`handlers.(Create|GetAll|Get|Update|Delete)Announcement` outside that file returns
nothing. So this is a full CRUD handler plus a migrated table with no endpoint: the
**mirror image of a phantom endpoint** (real server code that the API never exposes).

This is the same defect class the owner asked about, inverted: RES-014 was a frontend
call with no handler; this is a handler with no route.

### 1.3 Who creates, and who sees

**Create:** `POST /community/announcements` → `CreateCommunityAnnouncement`
(`routes.go:333`, `community_handler.go:447`).

- Role gate: `super_admin` or `unit_admin` only (`:470-473`); everyone else 403.
- A `unit_admin` may only target their own unit (`:484-489`).
- If no `unitId` is supplied, the author's own unit is used, and if the author has no
  unit the announcement is created **unit-less** (`:491-496`).
- `Type` defaults to `general` (`:508-510`), `Priority` to `normal` (`:511-513`).

**Read:** `GET /community/announcements` → `GetCommunityAnnouncements`
(`routes.go:334`, `community_handler.go:539`).

The filter is **`unit_id` only** (`:551-561`):

| Caller | Rows returned |
|---|---|
| `citizen` with a unit | `unit_id IS NULL OR unit_id = own` |
| `citizen` with **no** unit | `unit_id IS NULL` |
| `officer` with a unit | `unit_id IS NULL OR unit_id = own` |
| `unit_admin` with a unit | `unit_id IS NULL OR unit_id = own` |
| `super_admin` | **no filter at all** (`:551` branch not entered only for … see note) |

> Note on the `super_admin` case: there is no `super_admin` branch, and the `else-if`
> chain (`:551-561`) means a `super_admin` falls through with **no `unit_id`
> predicate**, so a super admin sees every published announcement platform-wide. That
> is probably intended for moderation, but it is worth stating plainly because it is
> implicit rather than written as a rule.

### 1.4 The geo-scope question — the important one

**Announcements do not inherit the feed's scope ladder at all.**

- `CommunityAnnouncement` has **no `GeoPlace` embed** (`models/Community.go:79-101`),
  whereas `ForumPost` does (`:34`). There are no `scope`, `state`, `lga` or `locality`
  columns on `community_announcements`.
- `GetCommunityAnnouncements` never calls `resolveViewerPlace`, `scopeWindowCondition`
  or `placeVisibleToViewer` (`:539-571`). Those live in `community_geo.go` and are
  used only by the forum-post paths.

Consequence: the entire `locality → lga → state → country → world` model
(D-011/D-013, `community_geo.go:17-38`) **does not apply to announcements.** The only
audiences that exist are "one specific unit" and "everyone". There is no way to
announce to an LGA or to a state, and no way to keep an announcement within the
viewer's own place.

### 1.5 `ExpiresAt` is write-only

`ExpiresAt` is accepted on create (`community_handler.go:454`, parsed `:498-506`,
stored `:523`) and returned to clients (`:49`) — but **never filtered on**:
`GetCommunityAnnouncements` constrains only `status = 'published'` (`:548`). No
scheduler job touches announcements (`backend/services/scheduler_service.go:95-307`
covers elections, invites, sessions, financial years, rankings, appeals,
officer-of-the-week, minor statuses, account purge). So an announcement that expired
last month is still served. `Status` is likewise never moved off `published`, so the
status predicate is a constant.

### 1.6 Does an admin announcement reach the feed?

**No — separate feed, and it does not flow through `GetForumPosts`.**

`GetForumPosts` queries `forum_posts` only (`community_handler.go:216-327`). The
frontend renders announcements from a different endpoint in a different tab
(`LiveCommunityPage.tsx:129` read, `:139` create, `:230` render). Therefore
announcements **do not** inherit:

- the geo-scope filter (§1.4),
- the keyset cursor (RES-014),
- `idx_forum_posts_scope_place` / `idx_forum_posts_feed_order`
  (`migrate.go:152-161`),
- the report endpoint (`POST /community/posts/:id/report` takes a **post** id; an
  announcement id is not a valid target — `community_moderation.go:45` looks up
  `ForumPost`).

There is **no index at all** on `community_announcements` for the unit filter, unlike
`forum_posts` which got two explicit indexes. Announcement listing is a sequential
scan filtered on `unit_id`.

---

## 2. Events

### 2.1 Model and ownership

| Fact | Evidence |
|---|---|
| `CommunityEvent` struct | `backend/models/Community.go:103-127` |
| Table `community_events` | `:125-127` |
| Migrated | `migrate.go:61` |
| Fields | `UnitID`, `Title`, `Description`, `Location` (free text), `Latitude`, `Longitude`, `EventDate`, `EndDate`, `Type`, `Status`, `MaxAttendees`, `AttendeeCount`, `AuthorID` |
| RSVP join table `EventAttendee` | `models/Community.go:137-151`, migrated `migrate.go:62` |

**Create:** `POST /community/events` → `CreateCommunityEvent` (`routes.go:335`,
`community_handler.go:574`). Role gate `super_admin` | `unit_admin` (`:601-604`);
`unit_admin` restricted to their own unit (`:615-621`); `Status` hardcoded to
`"upcoming"` (`:663`); `endDate` must be after `eventDate` (`:644-647`).

### 2.2 Who can see it — again unit-only, not scope-aware

`GetCommunityEvents` (`:680-711`) filters `status = 'upcoming'` (`:689`) plus exactly
the same `unit_id`-only branch chain as announcements (`:691-701`). `CommunityEvent`
has `Latitude`/`Longitude` but **no `GeoPlace` embed** and the handler never calls the
geo predicates. So:

- Events are unit-scoped, not place-scoped. An event cannot be addressed to an LGA or
  state, and an officer/citizen in a *neighbouring* unit in the same LGA cannot see it.
- This is the **opposite** of D-015 (the LGA convergence group), which deliberately
  spans units within an LGA. Events and the LGA group therefore contradict each other
  on what "local" means.

### 2.3 Lifecycle, cancellation, reminders — none exist

- **No cancellation, update or delete endpoint.** Searching `backend/` for
  `CancelEvent|DeleteCommunityEvent|UpdateCommunityEvent` returns nothing. The only
  two event routes are create and read (`routes.go:335-336`) plus RSVP (`:337`).
- **No lifecycle transitions.** `Status` is written once, as `"upcoming"`
  (`community_handler.go:663`), and nothing ever changes it. `GetCommunityEvents`
  filters on `status = 'upcoming'` (`:689`), a value that is therefore a constant —
  the predicate does nothing. A past event keeps being listed as upcoming forever.
- **No reminders.** The scheduler (`main.go:84-86`, job bodies
  `scheduler_service.go:95-307`) has no event job.
- **No notification on create or RSVP** — see §4.4.

Net: the `Status` field and the `EndDate` field are stored but functionally unused.
There is no way to cancel an event, and no way to see a past event.

### 2.4 RSVP path — RES-004 is wired; RES-008 is still open

`POST /community/events/:id/rsvp` → `RSVPToEvent` (`routes.go:337`,
`community_handler.go:714-808`).

**RES-004's concurrency fix is present and live**, not merely merged:

| Mechanism | Evidence |
|---|---|
| Row lock serialising concurrent RSVPs | `clause.Locking{Strength: "UPDATE"}` `:744-745` |
| Attendee count derived, not trusted | `:755-760` |
| Duplicate live RSVP rejected | `:764-772` + partial unique index `idx_event_attendee_active` (`migrate.go:141`, model tag `models/Community.go:139-141`) |
| Denormalised counter resynced after insert | `:787-792` |
| Failure modes mapped without a 500 | `:731-735`, `:796-807` |

**But the object-level authorization gap is still open — this is RES-008, which
remains `unclaimed`:**

- `First(&event, "id = ?", id)` (`:744-745`) loads the event with **no visibility
  check** and **no `status` check**. The handler never calls `placeVisibleToViewer` or
  `scopeWindowCondition`, and never reads `event.UnitID` to compare it to the caller's.
- Therefore **any authenticated user can RSVP to any event by UUID**, including an
  event belonging to another unit, and including one that is full, cancelled or in the
  past (the last two being moot only because nothing can produce them today — see
  §2.3).

### 2.5 The events response drops data the model stores

`communityEventResponse` (`community_handler.go:52-58`) returns `id`, `unitId`,
`title`, `description`, `location`, `eventDate`, `endDate`, `type`, `attendeeCount`,
`maxAttendees`, `createdAt`. It **omits `latitude`, `longitude` and `status`**, all of
which exist on the model (`models/Community.go:108-114`) and are stored on create
(`:657-663`). So an event's map position and its lifecycle state cannot be shown in
the UI even if they were populated.

### 2.6 Does the frontend Events page call endpoints that exist?

**Yes — all three.** The events surface is the `events` tab of `LiveCommunityPage`,
not a separate page:

| Frontend call | Backend route | Exists |
|---|---|---|
| `GET /community/events` (`LiveCommunityPage.tsx:130`) | `routes.go:336` | ✅ |
| `POST /community/events` (`:140`) | `routes.go:335` | ✅ |
| `POST /community/events/:id/rsvp` (`:141`) | `routes.go:337` | ✅ |

There is **no phantom** in the events path. The gap is the reverse: endpoints the UI
*wants* (cancel, past events, event detail) do not exist, so the UI simply has no such
control.

---

## 3. Discussions

### 3.1 Is there a separate Discussion model?

**No.** A discussion is `ForumPost` + `ForumReply`.

| Fact | Evidence |
|---|---|
| `ForumPost` | `backend/models/Community.go:31-52`, table `forum_posts` (`:54-56`) |
| `ForumReply` | `:59-72`, table `forum_replies` (`:74-76`) |
| Both migrated | `migrate.go:58-59` |
| `GeoPlace` embedded in `ForumPost` | `:34` (the only community model that has it) |

There is no `Discussion` type anywhere in `backend/models/`. The word "discussions" is
purely frontend UI copy for the `forum` tab (`LiveCommunityPage.tsx:13`, `:157`).

### 3.2 How a discussion starts, and how its place is derived

`POST /community/posts` → `CreateForumPost` (`routes.go:329`,
`community_handler.go:66-146`):

1. Client sends `title`, `content`, optional `category`, an audience `scope`, and
   optional `latitude`/`longitude` (`:67-75`).
2. `scope` defaults to `defaultScopeFor(resolveViewerPlace(...))` when omitted
   (`:110-113`) and is validated against the ladder (`:114-117`).
3. **The place is resolved server-side** (`:119`) and the publish is rejected if the
   caller's place cannot express the requested scope (`:120-125`). The client never
   sends place names — this is D-012 enforced.
4. The stored `GeoPlace` is built from the viewer's place plus the client's
   coordinates (`:129`, `newPlace`).

So an "LGA" post always carries a real LGA, and a caller with an unresolvable place
cannot create a local post at all.

### 3.3 Who can see it, by scope

`GetForumPosts` (`community_handler.go:216-327`):

- `scope` query selects the breadth, default `lga` (`:225-227`), validated `:228-233`.
- Non-super-admins get `scopeWindowCondition(place, maxRank)` (`:264-270`), i.e. the
  OR-window over the viewer's own locality/LGA/state/country built in
  `community_geo.go:178-199`. It can **widen** the viewer's own area but can never
  **move** to another place.
- A caller whose `UnitID` is set additionally keeps their own unit's posts in view
  (`:271`).
- Super admins (`Role == "super_admin"`) get no geographic predicate at all
  (`:264`).
- Paging is keyset over `(is_pinned, created_at, id)` (RES-014, `:148-214`, `:283`).

`GetForumPostByID` (`:329-376`) applies the same boundary through `postVisibleTo`
(`:353`) before returning the post and its replies.

### 3.4 How reply gating applies (D-014)

`POST /community/replies` → `CreateForumReply` (`routes.go:332`,
`community_handler.go:377-444`):

- Locked posts reject replies (`:409-412`).
- **Visibility = participation** (D-014): `postVisibleTo(userObj, post, lat, lng)`
  gates the reply (`:414-420`) — you cannot reply to a thread you could not open.

Two defects in this handler, both found while reading it:

1. **The `view_count` race RES-014 fixed is repeated here.** `:435-438` does
   `post.ReplyCount++`, `post.LastReplyAt = &now`, then `config.DB.Save(&post)` — a
   read-modify-write and a **full-row** write on every reply. RES-014 replaced exactly
   this pattern in `GetForumPostByID` with an atomic
   `UPDATE … SET view_count = view_count + 1`; the reply-count and `LastReplyAt`
   updates were not included in that fix. Concurrent replies to one thread can lose
   increments and clobber columns.
2. **The post is loaded without the `status` filter.** `:404` is
   `config.DB.First(&post, "id = ?", postID)`, whereas `GetForumPostByID` requires
   `"id = ? AND status = ?", postID, "published"` (`:345`). A post whose `Status`
   is not `published` is therefore invisible to the feed and to the detail endpoint but
   **still accepts replies**, provided it is geographically visible. Nothing in the
   current UI sets `Status` away from `published`, so this is a latent hole rather than
   an active one — but it is a real divergence between two endpoints that are supposed
   to enforce the same boundary.

---

## 4. UI/UX integration

### 4.1 Route → component map (community touches only)

The owner asked for `/community`, `/community/events`, `/community/discussions`.
**Only two of those routes exist**; the other two are *tabs inside one component*, not
routes.

| Path | Component | Role guard | Evidence |
|---|---|---|---|
| `/community` | `LiveCommunityPage` | `['citizen','super_admin']` | `App.tsx:150` |
| `/admin/community` | `LiveCommunityPage` (same component) | `ADMIN_ROLES = ['unit_admin','super_admin']` | `App.tsx:151`, `:72` |
| `/lga` | `LGAChannelPage` → `LGAChannelPanel` | **none** (authenticated only, via `ProtectedShell`) | `App.tsx:154`, `:137`; `LGAChannelPage.tsx:3,35` |
| `/news` | `NewsPage` | `['citizen']` | `App.tsx:148` |
| `/alerts`, `/alerts/:id` | `AlertsPage`, `AlertDetailPage` | `['citizen','super_admin']` | `App.tsx:146-147` |

**There is no `/community/events`, `/community/discussions` or `/community/moderation`
route.** `LiveCommunityPage.tsx:13` defines `type Tab = 'forum' | 'announcements' |
'events' | 'moderation'` and switches in-component (`:157`). Consequences:

- A discussion, announcement, event or moderation queue **cannot be deep-linked** —
  there is no URL state for the selected tab or the selected post. Refreshing returns
  you to the `forum` tab.
- The moderation queue added by RES-014 is reachable only by a role-gated in-page tab.

`/lga` having no `RequireRole` is **not** an anomaly: `/profile`, `/sos`, `/assistant`
and `/onboarding` are the same, and `RequireRole` is applied only where the role set
differs from "any authenticated user" (`RequireRole.tsx:13-55`). Server-side,
`/lga/channel` is behind `AuthMiddleware` (`routes.go:38-39`) and head-admin checks in
`lga_channel_handler.go`.

### 4.2 Role gating is client-side; officers have no community surface

`RequireRole` is a **frontend** route guard (`frontend/src/auth/RequireRole.tsx:13`).
`Role = 'citizen' | 'officer' | 'unit_admin' | 'super_admin'` (`types/api.ts:9`).

- `/community` admits `citizen | super_admin` — **not** `officer`, **not**
  `unit_admin`.
- `/admin/community` admits `unit_admin | super_admin`.
- An **`officer`** is admitted by neither, and `RequireRole.tsx:54-55` redirects an
  officer to `/officer/queue`.

Yet `GetCommunityAnnouncements` (`community_handler.go:557`) and `GetCommunityEvents`
(`:697`) both contain an explicit `officer` branch. Those branches are therefore
**unreachable from the UI** and only reachable by calling the API directly. Either the
community surface should include officers, or those branches are defensive-only and
should say so. The backend does not restrict these reads by role beyond
`AuthMiddleware`, so an officer calling the API directly *does* get their unit's
announcements and events.

### 4.3 Phantom-endpoint sweep — result

The owner asked to find any others of the RES-014 class. Method as in §0: every
frontend `api.<method>` call site (172 after excluding test files) was matched against
every registered route (299) with group-prefix and string-concatenation resolution.

**Result: no unmatched call sites.** The 8 initially reported were artefacts of
naive extraction, each hand-verified:

| Reported as unmatched | Actually resolves to | Exists at |
|---|---|---|
| `GET ${base}/officers` (×1) | `/sos/:id/responders/:unitId/officers` (`base` = `SosLiveTracking.tsx:18,54`) | `routes.go:231` |
| `PUT ${base}/officer` | `/sos/:id/responders/:unitId/officer` | `routes.go:232` |
| `POST ${base}/tracking` | `/sos/:id/responders/:unitId/tracking` | `routes.go:233` |
| `DELETE ${base}/tracking` | `/sos/:id/responders/:unitId/tracking` | `routes.go:234` |
| `PUT ${base}/location` | `/sos/:id/responders/:unitId/location` | `routes.go:235` |
| `POST /cases/:id/${action}` (`action ∈ dispatch\|arrive\|assign\|submit-review`, `useCases.ts:68`) | each action is a real route | `routes.go:161,164,165,168` |
| `GET /analytics/units/${unitId}` | `/analytics/units/:id` | `routes.go:381` |
| `POST /transfers/${id}/${action}` (`action ∈ approve\|reject`) | both real | `routes.go:273-274` |

**`POST /community/posts/:id/report` was the last genuine phantom and RES-014 closed
it.** The community surface now has **no** frontend call without a backend route.

**The community routes with no frontend caller: none.** Every route in the
`/community` block (`routes.go:327-351`) is called by at least one frontend site.

### 4.4 Notifications — there are none

No community handler creates a notification. Searching
`models.Notification{ | CreateNotification | notify` across
`community_handler.go`, `lga_channel_handler.go`, `community_moderation.go` and
`community_geo.go` returns **nothing** — while 14+ other handler files do create
notifications (`case_handler.go`, `sos_handler.go`, `alert_handler.go`, …).

Therefore, for all three of the audited features:

- a reply to your post notifies nobody;
- a new announcement notifies nobody;
- a new event notifies nobody;
- an RSVP notifies the organiser nobody;
- a **content report notifies the moderator nobody** — the queue added by RES-014 is
  pull-only; a moderator must remember to open the tab.

The report case is the most consequential: RES-014 made reports *work*, but nothing
tells a moderator that one arrived.

### 4.5 Moderation tab has no mock handler

`LiveCommunityPage.tsx:133` calls `GET /community/reports`, and `:151` calls
`POST /community/reports/:id/resolve`. `frontend/src/mocks/community.ts` defines
routes for posts (`:84,88,99`), replies (`:105`), report (`:115`), announcements
(`:123`), events (`:127`) and RSVP (`:132`) — but **not** `/community/reports` or the
resolve action. With `VITE_USE_MOCKS=true` (`mocks/config.ts:12`) the moderation tab
therefore has no mock backing and will error. Dev-only impact; recorded for
completeness.

---

## 5. Flow diagram — creation → scope → visibility → notification

```
ANNOUNCEMENT
  admin (super_admin|unit_admin only)          community_handler.go:470-473
      │  POST /community/announcements         routes.go:333
      │  unitId resolved: explicit → must equal own unit (:484-489)
      │                   omitted  → own unit, else NULL (:491-496)
      ▼
  community_announcements row                  models/Community.go:79-101
      │  NO GeoPlace. NO scope/state/lga/locality columns.
      │  ExpiresAt stored (:523) but never read by any query.
      ▼
  GET /community/announcements                 routes.go:334
      │  filter: unit_id IS NULL OR = own      :551-561
      │  super_admin: no filter at all
      │  NO geo predicate. NO cursor. NO index.
      ▼
  LiveCommunityPage "announcements" tab        LiveCommunityPage.tsx:129,230
      ▼
  NOTIFICATION: none                           §4.4

EVENT
  admin (super_admin|unit_admin only)          community_handler.go:601-604
      │  POST /community/events                routes.go:335
      │  Status hardcoded "upcoming"           :663
      ▼
  community_events row                         models/Community.go:103-127
      │  Lat/Lng stored (:657-659) but NOT in the response (:52-58)
      │  Status never transitions. No cancel route exists.
      ▼
  GET /community/events                        routes.go:336
      │  filter: status='upcoming' (constant)  :689
      │        + unit_id IS NULL OR = own      :691-701
      │  NO geo predicate. NO cursor. NO index.
      ▼
  LiveCommunityPage "events" tab               LiveCommunityPage.tsx:130,140
      ▼
  POST /community/events/:id/rsvp              routes.go:337 / handler :714-808
      │  row lock + derived count + unique idx  :744-792   ← RES-004, live
      │  NO visibility check, NO status check   ← RES-008, still open
      └─ event_attendees + attendee_count
      ▼
  NOTIFICATION: none                           §4.4

DISCUSSION
  any authenticated user (no role gate)        routes.go:329
      │  POST /community/posts {title,content,scope?,lat?,lng?}
      │  place resolved SERVER-side (D-012)     :119
      │  rejected if place can't express scope  :120-125
      ▼
  forum_posts row (GeoPlace embedded)          models/Community.go:31-52
      ▼
  GET /community/posts                          routes.go:330 / handler :216-327
      │  scopeWindowCondition(place,maxRank)    :264-270   ← geo ladder D-013
      │  + own unit's posts if UnitID set       :271
      │  keyset cursor (is_pinned,created_at,id) :283      ← RES-014
      │  indexed: idx_forum_posts_scope_place / feed_order  migrate.go:152-161
      ▼
  LiveCommunityPage "forum" tab                 LiveCommunityPage.tsx:119,157
      │  detail: GET /community/posts/:id       :126  (postVisibleTo :353)
      ▼
  POST /community/replies {postId,content}      routes.go:332 / handler :377-444
      │  IsLocked → 403                         :409-412
      │  postVisibleTo → 403                    :417     ← D-014: visibility = participation
      │  ⟂ BUG: post loaded WITHOUT status filter  :404  (§3.4-2)
      │  ⟂ BUG: ReplyCount++ then full-row Save    :435-438  (§3.4-1)
      ▼
  POST /community/posts/:id/report  (RES-014)   routes.go:349
      │  communityViewablePost → same D-014 boundary
      │  idempotent per (reporter, post)
      ▼
  reports row → GET /community/reports (pull-only queue)
      ▼
  NOTIFICATION: none                           §4.4

───── WHERE THE THREE SHARE CODE ─────
  shared: communityAuthor (:22) · auth via AuthMiddleware · unitId validation
          pattern (:89-104 / :475-496 / :606-626, copy-pasted three times)
  shared: page/cursor infrastructure — forum posts ONLY

───── WHERE THEY DIVERGE ─────────────
  geo ladder (GeoPlace, scopeWindowCondition, placeVisibleToViewer, cursor,
  indexes, report endpoint)  →  forum posts ONLY
  unit-only filtering        →  announcements + events
  role gate                  →  all three create paths: super_admin|unit_admin
  officer access             →  API branch exists, no UI route (§4.2)
  lifecycle/automation       →  none of the three has any
  notifications              →  none of the three has any
```

---

## 6. Findings, ordered by consequence

| # | Finding | Evidence | Class |
|---|---|---|---|
| 1 | **Reports notify no moderator.** RES-014 made reports work; the queue is pull-only and nothing signals arrival. | §4.4 | Functional gap |
| 2 | **Announcements are not geo-scoped.** `CommunityAnnouncement` has no `GeoPlace`; the filter is `unit_id` only. No LGA/state audience exists. | `models/Community.go:79-101`; `community_handler.go:551-561` | Design deviation from D-011/D-013 |
| 3 | **Events are not geo-scoped**, and contradict D-015 (LGA group spans units; events do not). | `models/Community.go:103-127`; `community_handler.go:691-701` | Design deviation |
| 4 | **RES-008 open: RSVP accepts any event by UUID** — no visibility, no status, no unit check. | `community_handler.go:744-745` | Authorization |
| 5 | **`CreateForumReply` repeats the RES-014 `view_count` race** — `ReplyCount++` + full-row `Save`. | `community_handler.go:435-438` | Concurrency |
| 6 | **`CreateForumReply` omits the `status='published'` filter** that `GetForumPostByID` enforces. | `:404` vs `:345` | Authorization (latent) |
| 7 | **`announcement_handler.go` is entirely unmounted** — full CRUD, migrated table, zero routes. | `announcement_handler.go:13,57,69,88,134`; absent from `routes.go` | Dead code |
| 8 | **`ExpiresAt` is write-only**; expired announcements served forever. No scheduler job. | `community_handler.go:548`; `scheduler_service.go:95-307` | Correctness |
| 9 | **No event lifecycle/cancellation/reminders.** `status='upcoming'` is a constant; past events stay "upcoming". | `community_handler.go:663,689`; no cancel route | Functional gap |
| 10 | **Officers have no community UI** although the API has officer branches. | `App.tsx:150-151`; `RequireRole.tsx:54-55`; `community_handler.go:557,697` | UI/API mismatch |
| 11 | **Event response drops `latitude`/`longitude`/`status`** — stored but unrenderable. | `community_handler.go:52-58` vs `models/Community.go:108-114` | Data loss at the edge |
| 12 | **No indexes on `community_announcements` / `community_events`** (forum posts got two). Unit-filtered listing is a seq scan. | `migrate.go:141-183` | Performance |
| 13 | **Moderation tab has no mock handler**; errors under `VITE_USE_MOCKS=true`. | `LiveCommunityPage.tsx:133,151`; `mocks/community.ts` | Dev-only |
| 14 | **Tabs are not routes** — discussions/events/moderation cannot be deep-linked; refresh resets to forum. | `LiveCommunityPage.tsx:13,157` | UX |
| 15 | **`unitId` validation logic duplicated three times** across the create handlers. | `:89-104`, `:475-496`, `:606-626` | Maintainability |

Explicitly **not** findings, checked and found consistent: `/lga` lacking `RequireRole`
(§4.1); all frontend community calls resolving to real routes (§4.3); the RES-004 RSVP
concurrency fix being live (§2.4).

---

## 7. What this audit did NOT establish

- **No runtime verification.** No Postgres on this box, so no request was executed
  against a real database. Every claim is static reading plus mechanical
  cross-checking.
- **Current database contents are unknown.** Whether any `community_announcements` or
  `community_events` rows exist in production, and how many have a NULL `unit_id`
  (i.e. are visible platform-wide), was not checked and is not assumed.
- **`docs/community/GEO-SCOPED-FEED.md` §6 citations** remain unverified; a second
  `websearch` attempt also returned nothing (recorded in the `RES-013` evidence row).
  Nothing in this document depends on that section.
- **No claim is made about other agents' uncommitted work.** `RES-006` (`antigravity`)
  and `RES-009` (`claude`) are unrelated to these files.
- **Not examined in depth:** `CommunityAlert`, `News`/`NewsAlert`, `ChatMessage`,
  `LGAMessage`, `Communication`, and the "communication room" membership authz — the
  first two are separate awareness surfaces, the rest are already filed under RES-015.

---

## 8. Suggested follow-ups (no code written; owner decides)

Ordered by consequence, not by effort. All would be **new rows**, not edits to RES-014
or RES-015 unless the owner prefers to fold them in.

1. **Notify moderators on a new report** (finding 1) — smallest change with the
   largest safety effect, and adjacent to RES-014's own subject matter.
2. **RES-008** (finding 4) — already filed and unclaimed; this audit confirms it is
   still real, and adds that the fix must also check `status`.
3. **`CreateForumReply` correctness** (findings 5 and 6) — one handler, two defects,
   both with a clear precedent to copy (`GetForumPostByID` for the status filter and
   RES-014 for the atomic increment).
4. **A decision on whether announcements and events should join the geo ladder**
   (findings 2, 3, 9) — this is the largest question and is a **product decision**, not
   a bug fix: making announcements geo-scoped changes who sees what, and it would
   change behaviour for existing rows. The owner should decide before any code.
5. **Retire or wire `announcement_handler.go`** (finding 7) — dead code with a live
   table; either mount it deliberately or delete both.
6. Housekeeping: event response fields (11), indexes (12), mocks (13), deep links (14),
   duplication (15).
