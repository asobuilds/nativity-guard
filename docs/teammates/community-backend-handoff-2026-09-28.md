# Community backend handoff — 2026-09-28

The frontend community review branch adds a read-only discussion detail view and prepares post/reply, admin announcement/event, and RSVP actions against the existing Go routes. **Live writes are disabled by default** (`VITE_COMMUNITY_WRITES_ENABLED` must explicitly be `true`). The branch changes frontend and documentation files only. This is the complete backend work list for the community surface, verified against `backend/routes/routes.go` and `backend/handlers/community_handler.go` on `main` at `af9ad1a`. Check the deployed service separately before enabling writes.

## Existing contract used by the frontend

| Action | Endpoint and request | Current response | Frontend use |
|---|---|---|---|
| Read posts/detail | `GET /community/posts`, `GET /community/posts/:id` | `{posts:[{id,title,content,author:{name},replyCount,...}]}`, `{post:{...,replies:[{content,author:{name},createdAt}]}}` | Live lists and detail with replies |
| Create post | `POST /community/posts` `{title,content}` | 201 `{post}` | Write flag only; note **`content`**, not mock `body` |
| Reply | `POST /community/replies` `{postId,content}` | 201 `{reply}` | Write flag only; note **`content`**, not mock `body` |
| Announcements | `GET /community/announcements`; admin `POST` `{title,content}` | `{announcements:[{title,content,publishedAt}]}` | Live list; admin form behind flag |
| Events | `GET /community/events`; admin `POST` `{title,description,location,eventDate,endDate,type:'meeting'}` with RFC3339 dates | `{events:[{eventDate,attendeeCount,...}]}` | Live list; admin form behind flag; both times required by current binding |
| RSVP | `POST /community/events/:id/rsvp` | 200 `{message}`; duplicate is 400 | Single-direction RSVP behind flag; no cancellation promised |

## Required backend fixes before enabling the flag

1. **Unit scope on list routes.** `GetForumPosts`, `GetCommunityAnnouncements`, and `GetCommunityEvents` apply a unit filter to officers and unit admins only if `User.UnitID` is present; an unassigned account can see all units' records. Reject unassigned staff or scope them to public content. Enforce a default-deny policy for unknown roles; cover citizen, assigned/unassigned officer/admin, and super-admin in tests. Confirm whether `User.UnitID` or active membership is the authoritative unit association.
2. **Post and reply validity.** `CreateForumReply` fetches a post by ID without checking `status='published'`; reject replies to removed/unpublished posts as well as locked posts. Reject malformed `unitId` in creation rather than silently creating a public post. Apply size constraints and rate limits server-side; HTML output is currently rendered as text by React, but the API must validate content independently.
3. **Event participation.** `RSVPToEvent` does not check event status, schedule or unit scope before registering a user; enforce visibility and eligibility. Make the capacity and attendee count update transactional/atomic so simultaneous RSVPs cannot exceed capacity. Prevent duplicate registrations with a database uniqueness constraint. Define a cancellation route if the product needs to support toggling; current frontend offers only one-way RSVP.
4. **Report and moderation workflow.** The mock `POST /community/posts/:id/report` route does not exist on `main`. Add authenticated report submission with reason, deduplication and rate limiting; a staff-only review queue and decision endpoint with audit logging; remove/hide status on moderated posts and replies; keep reports confidential from authors and unrelated users. Provide exact request/response shapes and role matrix so frontend can connect its report button and reviewer view.
5. **Announcement and event publishing.** Ensure `unit_admin` may publish only for an actively administered unit, including when `unitId` is omitted; reject invalid unit IDs. `GetCommunityAnnouncements` currently filters by status but not `expiresAt`; `GetCommunityEvents` filters by status `upcoming` but not an elapsed `eventDate`. Agree whether expired or past items should disappear or move to history and test the resulting list.
6. **Member state and paging.** Include the signed-in user's RSVP state in event listing/detail or provide a `my RSVPs` endpoint; frontend currently remembers a successful RSVP only for the current page session, and a reload could show the RSVP button again. Define pagination, limit and ordering for growing post, announcement and event feeds.

## Acceptance checks to coordinate

- Use dedicated test accounts for citizen, unit admin, super admin, assigned officer and officer with no unit; include cross-unit posts and events, locked and removed posts, full/expired events and simultaneous RSVP requests.
- Verify response shapes and create/read flow on a running Go instance with `VITE_USE_MOCKS=false`, then explicitly enable `VITE_COMMUNITY_WRITES_ENABLED=true` on a review deployment. Do not enable it in production until the permission tests and report/moderation decision are agreed.
- The frontend still has no production safety-tips source or community report UI. Agree editorial versus AI-sourced tips and their review/caching rules before presenting them as live safety advice.
