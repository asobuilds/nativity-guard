# Nativity Guard chat architecture

## Decision

Deliver two distinct chat products with a shared message-delivery foundation but separate room types, membership rules, and authorization checks:

1. **Citizen community chat** — geographic rooms at locality, LGA, and state scope.
2. **Security operations chat** — private unit and task-force rooms for verified security personnel.

A user must never gain access to a security room by being a citizen-room member, and citizen-room visibility must never expose security room metadata or messages.

## Existing capabilities to preserve

- The geographic community forum already resolves the caller's location and applies a server-side scope boundary.
- Unit communication rooms and task-force channels already exist.
- The WebSocket hub already supports room fan-out and presence.
- Existing content reports and community moderation provide reusable moderation concepts.

These are adjacent capabilities, not a complete citizen group chat. The new work must not relabel the forum or the LGA bulletin board as live chat.

## Citizen chat contract

- Room identity is derived from normalized geographic scope and place values; clients cannot choose another user's place boundary.
- A locality room requires a resolved locality and LGA/state; an LGA room requires LGA/state; a state room requires state.
- Membership is tied to the authenticated user's current verified/resolved place. A client-supplied room ID alone is never authorization.
- Room discovery returns only rooms in the caller's geographic boundary.
- Messages are persisted before broadcast, paginated newest-first with a stable cursor, and capped in length.
- User blocks prevent direct interaction and hide blocked-user content according to the server policy.
- Message reports enter a moderation queue; moderation actions are audited.
- Emergency announcements are a separate privileged message type and cannot be created by ordinary citizens.

## Security operations chat contract

- Security rooms are private and separate from citizen community rooms.
- Unit rooms require active membership in an active, verified unit.
- Task-force rooms require active task-force membership; officer access must be checked against current active enlistment.
- Super-admin visibility is explicit and audited; it must not silently widen ordinary unit access.
- Room IDs and task-force IDs supplied by clients are lookup keys only, never proof of membership.
- Reuse existing task-force channel identities where safe; do not weaken the existing walkie-talkie authorization checks.
- Security messages must not be included in public feeds, public search, or citizen room APIs.

## Shared message safeguards

- Persist and authorize every message on the server before broadcasting it.
- Enforce message-size, rate, and attachment limits server-side.
- Escape/render message content as text; do not accept arbitrary HTML.
- Validate attachment ownership, MIME type, size, and storage key before attaching it.
- Use soft deletion with an audit trail for moderation; do not silently erase evidence.
- Do not claim end-to-end encryption unless key management and client-side encryption are actually implemented and independently reviewed.
- Presence is ephemeral and must not expose precise user location.
- Retain only the minimum message metadata required by the product's published retention policy.

## Implementation sequence

1. Add dedicated citizen chat room, membership, message, block, and report persistence; add geographic authorization and REST history/send endpoints with tests.
2. Add citizen chat WebSocket delivery, server-derived room keys, rate limits, and presence.
3. Add citizen chat frontend room list, history, composer, report/block affordances, and empty/error states.
4. Integrate security text chat with the existing unit/task-force communication rooms and membership checks without changing PTT behavior.
5. Add security chat frontend surfaces, moderation/admin controls, end-to-end authorization tests, and CI verification.

## Release gates

- Cross-LGA and cross-state access tests must prove denial, not just successful same-area access.
- Citizen-to-security and security-to-citizen room access tests must prove isolation.
- Removed/expired unit or task-force membership must lose access immediately on the next request/connection.
- Message persistence, cursor pagination, rate limiting, moderation, and attachment authorization must be covered by tests.
- Backend tests, race tests, and frontend build must pass before merge.
