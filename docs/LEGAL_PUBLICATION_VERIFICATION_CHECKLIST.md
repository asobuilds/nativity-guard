# Nativity Guard — Public legal and privacy publication checklist

**Created:** 2026-10-23  
**Status:** Requires accountable operator/legal review before publication  
**Scope:** Public /about and /terms pages, Terms/Privacy endpoint, acceptance version, published support channels

This checklist is an internal release gate, **not a certificate, registration record, legal opinion, or assertion of compliance**. Never invent company details, certifications, NDPC registration, privacy contacts, government partnerships, or compliance status.

## Verified from source

- Public frontend routes `/about` and `/terms` exist without a sign-in guard.
- Backend `GET /api/v1/terms/:role` is a public API route with `?kind=terms|privacy`.
- The privacy query is resolved at role `all`; citizen Terms use role `citizen`.
- Backend seeds Terms and Privacy records on startup but published production data and uptime have **not** been independently checked.
- Backend returns the active document and its own version string in the Phase 3 patch.
- Published About page lists `support@nativityguard.com`; mailbox ownership, reachability, and response procedures are unverified.
- The `/feedback` UI is behind authentication, so anonymous visitors cannot use it directly.
- Publication is **not** evidence of actual contract validity, security effectiveness, or Nigerian data-protection compliance.

## Unverified operator facts: DO NOT PUBLISH AS ESTABLISHED UNTIL CONFIRMED

- [ ] Full legal person/entity name responsible for Nativity Guard
- [ ] Any company/business registration number, if applicable, checked against authoritative registration evidence
- [ ] Principal business/contact address if legally required, verified with accountable operator
- [ ] Data-controller identity and the contact route for privacy requests
- [ ] Data Protection Officer identity/contact if appointed or legally required (do not assume the title)
- [ ] Control of public domain(s) and the listed support mailbox, tested with real email delivery
- [ ] Rights to product name, trademarks, artwork, logos, copy, and included third-party assets
- [ ] Approval of final Terms and Privacy wording by a qualified Nigerian legal/privacy reviewer

## Privacy Notice content to verify against actual data processing

- [ ] Data collected (including location, evidence, identity documents, reports, contacts and account events)
- [ ] Specific purposes and lawful basis for each important processing activity
- [ ] Categories of people who may receive data, including participating units and service providers
- [ ] Processing of children's/teenagers' data; protections and age gates assessed
- [ ] Data storage countries, possible international transfers, vendors and safeguards
- [ ] Retention/deletion policies, including legally or operationally retained incident records
- [ ] Access, correction, deletion, objections, restrictions, and complaint/request routes
- [ ] Contact method and expected response handling; accessibility of public contact route
- [ ] Security language backed by deployed configuration, not merely design intentions
- [ ] Publication/effective date, document version, change history and consent version handling
- [ ] Any cookies/local-storage tracking and real privacy disclosures

## Deployment and QA gate

- [ ] Verify `https://nativityguard.com/terms#terms` and `https://nativityguard.com/terms#privacy` in an actual browser
- [ ] Verify `GET /api/v1/terms/citizen` returns an active, readable citizen terms document
- [ ] Verify `GET /api/v1/terms/citizen?kind=privacy` returns an active, readable privacy notice
- [ ] Confirm the API response version matches the **returned document version** and the version accepted during signup
- [ ] Confirm one legal endpoint failing still leaves the other document readable, with a retry action
- [ ] Confirm the About feedback link explains sign-in, and the support channel is accurate
- [ ] Verify narrow/mobile and keyboard navigation to legal anchors and sections
- [ ] Legal/privacy reviewer approves content before public compliance claims or submission of sensitive user data

### External official references for reviewer

- Nigeria Data Protection Commission: https://ndpc.gov.ng/
- NDPC FAQ: https://ndpc.gov.ng/faqs/
- Nigeria Data Protection Act 2023: https://ndpc.gov.ng/ndp-act-2023/

## Known access limits

Source review can verify code and route wiring but not conclusively verify `nativityguard.com` responses, mailbox delivery, registered operator identity, data flows, or legal compliance. The live URLs and policy API were inaccessible through the available web inspection tool at the time this checklist was written. All these checks remain open until a responsible owner provides evidence and browser/API acceptance passes.
