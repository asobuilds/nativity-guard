# Community: geo-scoped feed, comment gating and the LGA convergence group

Status: design + implementation plan, 2026-10-18.
Owner: `opencode` (RES-005, RES-012, RES-013).
Grounded in a direct read of the code as of `main@3db0885`. No claim below is
based on a summary — each finding cites the file and symbol it comes from.

---

## 1. Why this document exists

The owner's requirement, in one paragraph: community feeds must be **local**.
A user sees posts from their village/locality, their LGA, their state, their
country, and the world — and nothing else by default. Users must not be able to
comment on posts from places they are not part of, so a Lagos resident cannot
argue in a Benue village thread. Separately, every **security unit registered in
an LGA** should be able to converge in one group for that LGA: all units in the
LGA can join and read, but **only head admins may post**. Everything must be
wired live — no demo data, no placeholder toggles.

The brief also asked for a grounded design inspired by a **hybrid of X and
Facebook** for a national security community service. Section 6 does that
honestly, separating what is well-established product mechanics from what is
speculation.

---

## 2. What exists today (verified)

### 2.1 The forum has no geography at all

`models.ForumPost` (`backend/models/Community.go:10-34`) has `UnitID`, `Title`,
`Content`, `Category`, `AuthorID`, `Status`, pins, counts, timestamps. There is
**no** scope, state, LGA, locality, coordinate or geohash column. `ForumReply`
(`:37-50`) has `PostID`, `Content`, `AuthorID`, `IsSolution`, `Status` — also no
geography.

`GetForumPosts` (`backend/handlers/community_handler.go:119-156`) filters only by
`unit_id` (nullable) and role, then `ORDER BY is_pinned DESC, created_at DESC`
with **no limit and no pagination**. A citizen with no unit gets
`unit_id IS NULL` — i.e. every globally-authored post. So today a citizen in
Lagos and a citizen in Benue read the same feed. This is the core defect.

`CreateForumPost` (`:57-116`) takes `unitId`, `title`, `content`, `category`.
`CreateForumReply` (`:198-263`) checks `IsLocked` and the unit rule, nothing
else. `GetForumPostByID` (`:159-195`) checks the unit rule only.

**Consequence:** there is no path by which a post can be local, and no check
that would stop a cross-locality reply.

### 2.2 The geo *infrastructure* already exists and works

This is the good news and the reason this is a feed change, not a
geo-from-scratch project:

- **Reverse geocoding** — `services.GeocodingService.ReverseGeocode(lat, lng)`
  (`backend/services/geocoding_service.go:113`) returns a `models.GeocodeCache`
  (`models/GeocodeCache.go:8-25`) with `Suburb`, `LGA`, `State`, `Country`,
  `PostalCode`, `Landmark`. It is **cached by geohash precision 6 (~1.2 km)** for
  30 days, negative-cached for 24 h, serialised behind a global mutex to respect
  Nominatim's 1 req/s policy, and fails soft (returns a record with empty
  address fields rather than erroring). Calling it server-side per post/reply is
  cheap because of the cache.
- **Geohash encoding** — `utils.EncodeGeohash`.
- **Administrative dataset** — `backend/data/nigeria.json`: 36 states + FCT and
  all 774 LGAs. **No wards and no villages.** So the finest official level we can
  name is the LGA; "locality/village" must come from the geocoder's
  `Suburb`/`Village`/`Hamlet`/`Neighbourhood` (`pickAddress` already computes
  `Suburb` from `Suburb|City|Town|Village` and `Landmark` from
  `Amenity|Shop|Neighbourhood|Hamlet`). We accept that and document it.
- **Units carry place** — `models.SecurityUnit` (`models/Unit.go:9-39`) has
  `State` (indexed), `LGA` (indexed), `Ward`, `Latitude`, `Longitude`,
  `OperationalRadius`, `IsVerified`.

### 2.3 The user has no place of their own

`models.User` has **no** village/ward/LGA/state/coordinate fields. A staff
member's place is *derived*: `User.UnitID` → `SecurityUnit`. Citizens have no
unit, so they currently have no derivable place at all. Any "feed by location"
must therefore resolve the viewer's place from one of:

1. their verified unit (staff), or
2. a reverse-geocoded coordinate they supply with the request (citizens), or
3. their most recent stored location (`UserLocation`), if sharing is on.

### 2.4 The LGA channel exists but lets everyone post

`callerLGAContext` (`backend/handlers/lga_channel_handler.go:27-62`) walks
`UnitMembership{status: active}` → `SecurityUnit`, requires `IsVerified`, a
non-empty `LGA` and `State`, and returns `(lga, state, unitID, …)`. It does
**not** look at `IsHeadAdmin`, even though `UnitMembership` has that field
(`models/UnitMembership.go:37`).

`GetLGAChannel` (`:65-94`) reads the last 100 non-expired `LGAMessage` rows for
`(state, lga)`. `PostLGAChannel` (`:100-191`) validates category/priority/length,
rate-limits to 5 posts/hour/user, then creates the message. **Any verified member
can post.** The requirement is head-admins-only. There is also no membership
record for "joining" the LGA group and no member roster.

### 2.5 Live-wiring defects found during this scan

These are the "everything must be wired live" items; each is verified:

| # | Finding | Evidence |
|---|---|---|
| L1 | Community **writes are disabled in the UI by env gate** and gated behind a banner | `frontend/src/pages/LiveCommunityPage.tsx:17-18,47,51,55` (`VITE_COMMUNITY_WRITES_ENABLED`, default `'false'`) |
| L2 | The feed endpoint is called with **no pagination** — the whole table is loaded | `community_handler.go:148`; client `LiveCommunityPage.tsx:35` |
| L3 | A **report/moderation endpoint the frontend expects is not mounted** in Go | recon of `backend/routes/`; `LiveCommunityPage.tsx` has no report call wired to a live route |
| L4 | `ChatMessage` model exists but has no live route/handler (dead model) | `backend/models/` |
| L5 | LGA channel **read has no pagination beyond a hard 100** and no "load older" | `lga_channel_handler.go:80` |
| L6 | `GetForumPostByID` increments `view_count` with a full-row `Save`, racing under concurrency | `community_handler.go:189-190` |

L3–L6 are backlog (RES-014); L1/L2 are fixed with RES-005.

---

## 3. Design I: the geographic scope ladder

Ordered, most specific first. Stored as a short string.

| Rank | Scope value | Meaning | Match key |
|---|---|---|---|
| 0 | `locality` | village / ward / neighbourhood | `state` + `lga` + `locality` |
| 1 | `lga` | local government area | `state` + `lga` |
| 2 | `state` | state or FCT | `state` |
| 3 | `country` | national (Nigeria) | `country` |
| 4 | `world` | everything | — |

**Rule:** LGA names repeat across states (there are LGAs with the same name in
different states), so an `lga` match is always `state AND lga`. This is the
single most important correctness detail in the design.

A post stores the scope it is *addressed to*, plus the place it was authored at:

```go
type GeoPlace struct {
    Scope     string  // locality|lga|state|country|world
    Locality  string  // village/ward/suburb
    LGA       string
    State     string
    Country   string  // default "Nigeria"
    Latitude  float64
    Longitude float64
    Geohash   string  // precision 5 (~4.9 km) for coarse maps, not for matching
}
```

A viewer has a place `V` (same shape, without a scope — `V`'s scope is what they
are browsing). The **visibility predicate** for a post `P` is:

```
P.Scope == 'world'                                              -> yes
P.Scope == 'country' AND P.Country == V.Country                 -> yes
P.Scope == 'state'   AND P.State   == V.State                   -> yes
P.Scope == 'lga'     AND P.LGA     == V.LGA AND P.State == V.State -> yes
P.Scope == 'locality'AND P.Locality== V.Locality AND P.LGA == V.LGA
                     AND P.State   == V.State                   -> yes
otherwise                                                       -> no
```

### The feed algorithm

`GET /community/posts?scope={locality|lga|state|country|world}&lat=&lng=&limit=&offset=`

- `scope` is the **breadth the viewer asked for** — it is a request to *widen*,
  never a way to see somewhere else. Default `lga`.
- The server resolves `V` (viewer's place) and returns posts visible to `V`
  whose own scope rank is **≤** the requested rank. So `scope=state` returns
  locality + LGA + state posts from *the viewer's own* state; it never returns
  another state's posts, because the predicate pins the place to `V`.
- Ordering: `is_pinned DESC, created_at DESC`. Pagination: keyset-friendly
  `limit` (cap 50, default 20) + `offset`; `reply_count`/`last_reply_at`
  already exist for ordering later.
- Each returned post carries `scope` and its place so the UI can badge locality /
  LGA / state / national / world.

This is efficient: one indexed `WHERE` with an OR over four place branches plus
a scope-rank bound, served by a composite index (see §5). No N+1, no per-row
geocoding. Geocoding happens only on **write** (once per distinct ~1.2 km cell,
then cached).

### Where the viewer's place comes from (server-derived, never client strings)

```
resolveViewerPlace(ctx):
  1. verified active UnitMembership -> SecurityUnit{State,LGA,Ward}   (staff)
  2. else reverse-geocode (lat,lng) from the query/body               (citizen)
  3. else most recent UserLocation -> GeocodeCache                    (fallback)
  4. else empty place -> feed returns only `world` posts
```

The client may send **coordinates** and a **scope**. It may never send
`state`/`lga`/`locality` strings; those are always the server's own resolution.
This mirrors the existing, correct `callerLGAContext` pattern.

---

## 4. Design II: comment gating and the LGA convergence group

### Comment gating (reduces noise and abuse)

`CreateForumReply` runs the **same visibility predicate** on `(V, post)` before
inserting. If the post is not visible to the viewer they get `403` with a clear
message. This is a direct implementation of "users cannot comment on things of
other places": visibility and participation are the same boundary, so there is
no second rule to keep in sync. `GetForumPostByID` uses the same predicate, so a
post you cannot see is also a post you cannot open or count.

### LGA convergence group

The existing `lga_messages` channel is 90% of this already — per-`(state, lga)`
read for verified unit members. Two additions close the gap:

1. **Membership + roster.** New `LGAGroupMember{State, LGA, UnitID,
   JoinedByUserID, JoinedAt}` with a unique `(state, lga, unit_id)`. Endpoints:
   - `POST /community/lga-group/join` — the caller's verified unit joins the
     group for the LGA it is registered in (LGA/state taken from the unit, never
     the body).
   - `POST /community/lga-group/leave`
   - `GET  /community/lga-group/members` — roster (unit name, joinedAt).
   Reading the channel requires the caller be a verified unit member **in that
   LGA**; joining is what makes the group explicit and gives a roster.
2. **Head-admin-only posting.** `callerLGAContext` additionally returns
   `isHeadAdmin` (from `UnitMembership.IsHeadAdmin`, falling back to the unit's
   role). `PostLGAChannel` returns `403` unless `isHeadAdmin`. Read stays open to
   every verified unit member in the LGA. This is the requirement exactly:
   *all units in the LGA can see; only head admins create posts.*

---

## 5. Schema and index plan

`forum_posts` gains the embedded `GeoPlace` columns and:

```sql
CREATE INDEX idx_forum_posts_scope_place
  ON forum_posts (status, scope, state, lga, locality, created_at DESC);
CREATE INDEX idx_forum_posts_feed_order
  ON forum_posts (status, is_pinned DESC, created_at DESC);
```

`lga_group_members` gets a unique index on `(state, lga, unit_id)`.

GORM `AutoMigrate` adds columns and indexes; the tagged unique index on
`LGAGroupMember` is the backstop, exactly like `EventAttendee`'s partial index
(`models/Community.go:108-129`) — and, following that precedent, a pre-existing
duplicate is a loud failure, not something silently papered over.

Backfill: existing posts have no place and will therefore only appear under
`world`. That is the honest default (a post of unknown place must not be
presented as local). A migration note records this.

---

## 6. Research: an X × Facebook hybrid for a security community service

The brief asked for grounding. Web search was unavailable in this environment
and direct fetches returned only navigation chrome, so this section is
**explicitly labelled as design reasoning from well-established product
mechanics**, not cited sources. Treat it as a hypothesis set to validate with
users, not as evidence.

**Facebook contributes the group and audience mental model.** The load-bearing
idea is the *audience selector*: an author chooses who a post is for, and the
platform refuses to show it to anyone else. That is precisely the scope ladder
in §3 — the difference is that Facebook's audiences are social graphs, ours are
administrative places. Facebook Groups also supply the convergence-group pattern
in §4: a bounded space with membership, a roster, and a posting right that is not
the same as the read right. Both map cleanly onto an LGA.

**X contributes the real-time, ranked, one-feed pattern.** X's mechanics that
transfer are: a single chronological-or-ranked timeline rather than many forums;
pinning and reply counts as cheap relevance signals (already present on
`ForumPost`); and a strict, cheap participation gate. What does **not** transfer
is open reply-for-all and follower-based distribution — in a security service,
reach must be bounded by jurisdiction, not by popularity. Our feed is therefore
*reach-bounded first, ranked second* (rank only reorders what is already
visible).

**The hybrid, stated plainly:** locality-first distribution (Nextdoor-style
hyperlocalism, which this design adopts without its invite graph) + audience
scoping and groups (Facebook) + a real-time chronological feed with pinned and
reply-count signals (X) + a hard jurisdictional participation gate enforced
server-side (neither).

Features to road-map (RES-014 and later), ranked by value/effort:

1. **Report + moderation queue** (L3) — closes the abuse loop the gate opens.
2. **Keyset pagination** with `last_reply_at` (L2, L5) — needed before growth.
3. **Scope-aware notifications** — only notify a user about posts inside their
   visible window; reuse the existing LGA/State notification machinery.
4. **Unit convergence beyond the LGA** — state-level and multi-LGA task forces,
   by generalising `LGAGroupMember` to a `(scope, key)` membership.
5. **Ranking** — a small, explainable score (`pin + recency + reply velocity`)
   applied *after* the visibility filter; never as a way to escape it.

---

## 7. Phased plan and board mapping

| Phase | Row | Scope |
|---|---|---|
| 1 | RES-005 | `GeoPlace` on `ForumPost`; server-derived place; scope-filtered, paginated feed; visibility predicate on detail and replies; remove the writes gate; frontend scope selector + audience composer + reply gating |
| 2 | RES-012 | LGA convergence group: join/leave/roster; head-admin-only posting; frontend group UI |
| 3 | RES-013 | This document (research + architecture) |
| 4 | RES-014 | Live-wiring scan fixes: report+moderation queue, keyset pagination, scope-aware notifications, ranking, `ChatMessage` dead model, `view_count` race |

Decisions recorded in `resolution.md`: D-011 (scope ladder), D-012 (server-derived
place), D-013 (strict window + State-paired LGA matching), D-014 (visibility =
participation), D-015 (LGA group: read for all members, post for head admins),
D-016 (go live: remove the env gate).
