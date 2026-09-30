package content

// TermsVersion is the current published version of every document below.
// Bump this string whenever ANY document changes, then update the content.
// Previous versions remain in the database and are still linked from the
// acceptance rows that referenced them.
const TermsVersion = "2026-09-30-v1"

// UniversalTerms is the master text that applies to every role. The
// role-specific section (RoleTerms) is appended when the service seeds
// a document.
//
// DRAFT — pending legal review.
const UniversalTerms = `# Nativity Guard — Terms of Service (DRAFT — pending legal review)

**Version: ` + TermsVersion + `**
**Effective: 30 September 2026**

## 1. What Nativity Guard is

Nativity Guard is a coordination platform. It connects citizens with
independent community security units. It is not a security provider, an
emergency service, a police force, or a dispatch agency.

If you need emergency help, contact your local emergency number. A report
or SOS sent through this platform may be delayed, may not be received,
and does not guarantee a response.

## 2. Developer non-interference and release

Nativity Guard, its developers, operators, officers, employees and
affiliates (together "the Operators"):

- Do not direct, supervise, train, employ, or control any security unit
  that responds to activity on this platform.
- Do not investigate, verify, or guarantee the accuracy, completeness,
  or legality of any case report, SOS alert, or other content submitted
  by any user.
- Do not influence, approve, or take part in any tactical, operational,
  or investigative decision made by any security unit.
- Do not dispatch, direct, or coordinate the physical response of any
  security unit.

All operational decisions are the sole responsibility of the responding
unit and are made independently of the platform.

You release the Operators from every past, present and future claim,
demand, or dispute arising from or connected to the actions, omissions,
delays, or outcomes of any security unit, any other user, or any third
party using or interacting with this platform — including any harm,
loss, injury, or failure to respond.

## 3. Acceptable use

You agree not to submit knowingly false information, impersonate another
person, harass any user, or misuse this platform for any purpose other
than the genuine community-safety purposes it was built for.

## 4. Account, suspension and removal

We may suspend, restrict, or permanently remove your account if you
breach these terms, misuse the platform, or place other users at risk.
Where a suspension has a fixed duration, it will lift automatically. A
permanent ban requires a super administrator's decision and is logged.

## 5. Governing law

These terms are governed by the laws of the Federal Republic of Nigeria.

## 6. Changes

We may update these terms. Material changes take effect when a new
version is published and you are asked to accept it again. Continued
use of the platform after a new version is published means you accept
the new version.
`

var RoleTerms = map[string]string{
    "citizen": `## Your responsibilities as a citizen

- Report incidents truthfully and to the best of your knowledge.
- Do not use the platform to publish unverified allegations.
- Respect the privacy of victims, witnesses, and suspects.
- You can withdraw location consent at any time in Settings; doing so
  means reports you file from then on will not carry your coordinates.
`,
    "officer": `## Your responsibilities as an officer

- Access only the cases assigned to you or to your unit.
- Record progress, evidence and outcomes accurately and promptly.
- Protect evidence and follow your unit's approval and escalation
  procedures.
- Do not export, share, or alter case records for any reason other than
  your assigned duties.
- Treat AI suggestions as suggestions; verify independently before
  acting.
`,
    "unit_admin": `## Your responsibilities as a unit administrator

- Grant and review access only for legitimate unit work.
- Review case decisions, assignments, and public notices through the
  authorised workflows.
- Maintain an audit trail; do not use administrator access to bypass
  permissions.
- Protect the privacy of reporters, officers, and the wider public.
`,
    "super_admin": `## Your responsibilities as a platform administrator

- Use platform-wide access only for approved operational, security, and
  governance purposes.
- Keep role changes, suspensions, bans, and other sensitive actions
  accountable through the audit log.
- Investigate misuse fairly and consistently across units.
- Apply the same privacy and evidence protections to every account,
  including your own.
`,
}

// PrivacyContent — NDPR requires that data-processing consent is separate
// from general terms acceptance, so this is stored as its own document.
//
// DRAFT — pending legal review.
const PrivacyContent = `# Nativity Guard — Privacy Notice (DRAFT — pending legal review)

**Version: ` + TermsVersion + `**
**Effective: 30 September 2026**

## What we collect

- Account details you provide: name, email, phone, date of birth, password.
- Content you submit: reports, SOS alerts, evidence, comments.
- Location data: coordinates from your device when you grant permission,
  attached to reports and SOS alerts you submit.
- Technical data: IP address, user agent, session records.

## Why we collect it

- To operate the platform: create accounts, show cases to authorised
  users, deliver notifications.
- To coordinate emergency responses: attach location and contact details
  to SOS alerts so responding units can act.
- To meet legal obligations: keep an audit trail of administrative
  actions.

## Who can see it

- Officers and unit administrators see the cases assigned to their unit.
- Platform administrators see what is necessary for governance.
- Other citizens see only the public fields of reports marked public.

## Your rights (NDPR)

You have the right to access, correct, and request deletion of your data.
Withdrawing consent stops future processing; it does not erase records
already lawfully processed. To exercise any right, contact the platform
administrator through the support channel.

## Retention

- Case and evidence records: 5 years, then archived.
- Avatars and covers: while your account exists, then 30 days after
  deletion.
- Audit logs: 5 years.

## Contact

Data controller: the Nativity Guard platform operator. Contact details
will be published on the About page.
`