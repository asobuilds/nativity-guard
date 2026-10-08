import { Link } from 'react-router-dom'
import {
  ArrowRight,
  CheckCircle2,
  Eye,
  FileText,
  MapPin,
  Shield,
  ShieldCheck,
  UserCheck,
} from 'lucide-react'
import { CASE_STATUS_META, FIELD_LIFECYCLE, REVIEW_PHASE } from '@/lib/status'

/**
 * The front door — `/`, seen by someone who is not signed in.
 *
 * Everything else in this app assumes a person who is already inside. This is the
 * only screen a stranger sees, and it is written for a resident who has never heard
 * of Nativity Guard, arrived from a neighbour's WhatsApp forward, and is deciding in
 * seconds whether this is a real service.
 *
 * Three rules from `frontagent` §11 shape it:
 *
 *  - **Nothing here is claimed that the code does not do.** The accountability
 *    block states the three rules that are actually enforced server-side (they are
 *    cited in the comments below). A landing page that oversells is the first lie
 *    the product tells.
 *  - **No manufactured urgency.** No countdowns, no "N people are unsafe right
 *    now", no notification prompt before the product has earned it.
 *  - **No invented data.** This screen makes no API calls at all — see the note on
 *    the map block. Nothing on it can go stale, and a cold backend cannot make it
 *    look broken.
 */
export function LandingPage() {
  return (
    <div className="ng-public-scene min-h-screen bg-base">
      <header className="glass-bar relative z-10 mx-auto flex max-w-5xl items-center justify-between rounded-b-panel px-6 py-5 backdrop-blur-xl">
        <div className="flex items-center gap-2">
          <Shield className="size-6 text-signal" aria-hidden />
          <span className="text-sm font-bold tracking-wide text-ink">NATIVITY GUARD</span>
        </div>
        <Link
          to="/auth/login"
          className="text-sm text-ink-muted transition-colors hover:text-ink"
        >
          Sign in
        </Link>
      </header>

      <main className="relative z-10 mx-auto max-w-5xl px-6">
        {/* ------------------------------------------------------------ hero */}
        <section className="dawn-hero my-4 rounded-panel border border-border-hi px-6 py-14 sm:px-10 sm:py-20">
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] text-ink-muted ring-1 ring-border-hi">
            <ShieldCheck className="size-3.5 text-signal" aria-hidden />
            Community safety, on the record
          </span>

          <h1 className="mt-5 max-w-2xl text-3xl font-semibold leading-tight text-ink sm:text-4xl">
            Every report answered. Every case accounted for.
          </h1>

          <p className="mt-4 max-w-xl text-sm leading-relaxed text-ink-muted sm:text-base">
            Nativity Guard connects residents, security units and administrators on one shared
            record — so a report you file does not disappear into a phone call.
          </p>

          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Link
              to="/auth/signup"
              className="inline-flex h-12 items-center gap-2 rounded-lg bg-signal px-5 text-base font-semibold text-signal-ink transition-colors hover:bg-signal/90"
            >
              Get started
              <ArrowRight className="size-4" aria-hidden />
            </Link>
            <Link
              to="/auth/login"
              className="inline-flex h-12 items-center rounded-lg border border-border-hi bg-surface-hi px-5 text-base text-ink transition-colors hover:bg-surface-hi/80"
            >
              Sign in
            </Link>
          </div>

          <p className="mt-4 text-xs text-ink-faint">
            Free to use. You must be 16 or older to hold an account.
          </p>
        </section>

        {/* ------------------------------------------------------- lifecycle */}
        <section className="border-t border-border py-14">
          <h2 className="text-xl font-semibold text-ink">What happens after you report</h2>
          <p className="mt-2 max-w-xl text-sm text-ink-muted">
            Not a promise — the actual sequence a case moves through, and the point at which someone
            else has to agree before it can be called finished.
          </p>

          {/* The field stages are one-way; the review phase is a loop. Drawing them
              as a single rail would misrepresent the second half (`frontagent` §8),
              so they are two separate groups with the cycle drawn as one. */}
          <ol className="mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            {FIELD_LIFECYCLE.map((status, index) => {
              const meta = CASE_STATUS_META[status]
              return (
                <li key={status} className="ng-public-glass rounded-panel border border-border bg-surface p-4">
                  <span className="flex items-center gap-2">
                    <span className="grid size-5 place-items-center rounded-full bg-surface-hi text-[11px] text-ink-muted tabular">
                      {index + 1}
                    </span>
                    <span className="text-sm font-medium text-ink">{meta.label}</span>
                  </span>
                  <p className="mt-2 text-xs leading-relaxed text-ink-muted">{meta.description}</p>
                </li>
              )
            })}
          </ol>

          <div className="mt-4 ng-public-glass rounded-panel border border-border-hi bg-surface p-5">
            <p className="flex items-center gap-2 text-sm font-medium text-ink">
              <UserCheck className="size-4 text-signal" aria-hidden />
              Then it has to be approved
            </p>
            <p className="mt-2 max-w-2xl text-xs leading-relaxed text-ink-muted">
              The officer who worked the case submits it for closure. An administrator either
              approves it or sends it back with a reason — and a sent-back case returns to the same
              review, not to the start. That back-and-forth is recorded, so "closed" means someone
              accountable agreed, not that someone stopped working.
            </p>
            <div className="mt-4 flex flex-wrap items-center gap-2">
              {REVIEW_PHASE.map((status, index) => (
                <span key={status} className="flex items-center gap-2">
                  {index > 0 ? (
                    <span className="text-xs text-ink-faint" aria-hidden>
                      →
                    </span>
                  ) : null}
                  <span
                    className={`rounded-full px-2.5 py-0.5 text-[11px] ring-1 ${CASE_STATUS_META[status].bg} ${CASE_STATUS_META[status].text} ${CASE_STATUS_META[status].ring}`}
                  >
                    {CASE_STATUS_META[status].label}
                  </span>
                </span>
              ))}
              <span className="text-xs text-ink-faint">
                — with changes, it loops back to review
              </span>
            </div>
          </div>
        </section>

        {/* --------------------------------------------------- what it is for */}
        <section className="border-t border-border py-14">
          <h2 className="text-xl font-semibold text-ink">What you can do with it</h2>

          <div className="mt-8 grid gap-4 sm:grid-cols-3">
            {[
              {
                icon: <FileText className="size-4" aria-hidden />,
                title: 'Report an incident',
                body: 'Describe what you saw, mark where it happened, attach what you have. You get a tracking reference straight away.',
              },
              {
                icon: <Eye className="size-4" aria-hidden />,
                title: 'Follow what happens to it',
                body: 'Watch the status change as a unit picks it up, dispatches and investigates. You do not have to call anyone to ask.',
              },
              {
                icon: <MapPin className="size-4" aria-hidden />,
                title: 'Know who covers your area',
                body: 'See which units operate near you and what area each one is responsible for.',
              },
            ].map((item) => (
              <div key={item.title} className="ng-public-glass rounded-panel border border-border bg-surface p-5">
                <span className="grid size-9 place-items-center rounded-lg bg-surface-hi text-signal">
                  {item.icon}
                </span>
                <p className="mt-3 text-sm font-medium text-ink">{item.title}</p>
                <p className="mt-1.5 text-xs leading-relaxed text-ink-muted">{item.body}</p>
              </div>
            ))}
          </div>
        </section>

        {/* ---------------------------------------------------- accountability */}
        <section className="border-t border-border py-14">
          <h2 className="text-xl font-semibold text-ink">
            The rules that make it worth trusting
          </h2>
          <p className="mt-2 max-w-xl text-sm text-ink-muted">
            These are not aspirations. They are enforced by the system, and each one is there because
            the alternative is a record nobody can rely on.
          </p>

          <ul className="mt-8 grid gap-4 sm:grid-cols-3">
            {[
              {
                title: 'Closure is an approval, not a button',
                body: 'The officer who worked a case cannot mark it finished. It goes to an administrator, who approves it or sends it back.',
              },
              {
                title: 'Nobody signs off their own work',
                body: 'An officer assigned to a case is blocked from approving the closure of that same case. Two people, always.',
              },
              {
                title: 'Unit access is not case access',
                body: 'A unit administrator sees the cases their unit handles. Only platform oversight sees across units — not by default, and not quietly.',
              },
            ].map((item) => (
              <li key={item.title} className="ng-public-glass rounded-panel border border-border bg-surface p-5">
                <CheckCircle2 className="size-4 text-ok" aria-hidden />
                <p className="mt-3 text-sm font-medium text-ink">{item.title}</p>
                <p className="mt-1.5 text-xs leading-relaxed text-ink-muted">{item.body}</p>
              </li>
            ))}
          </ul>
        </section>

        {/* ------------------------------------------------------------- close */}
        <section className="border-t border-border py-14">
          <div className="ng-public-glass rounded-panel border border-border-hi bg-surface p-8 text-center">
            <h2 className="text-xl font-semibold text-ink">
              Report something. Watch it get answered.
            </h2>
            <p className="mx-auto mt-2 max-w-md text-sm text-ink-muted">
              Creating an account takes a minute. You will need your email address and a phone number.
            </p>
            <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
              <Link
                to="/auth/signup"
                className="inline-flex h-12 items-center gap-2 rounded-lg bg-signal px-5 text-base font-semibold text-signal-ink transition-colors hover:bg-signal/90"
              >
                Get started
                <ArrowRight className="size-4" aria-hidden />
              </Link>
              <Link
                to="/auth/login"
                className="inline-flex h-12 items-center rounded-lg border border-border-hi bg-surface-hi px-5 text-base text-ink transition-colors hover:bg-surface-hi/80"
              >
                Sign in
              </Link>
            </div>
            <p className="mx-auto mt-5 max-w-md text-xs text-ink-faint">
              We ask for your date of birth to confirm you are old enough to hold an account. It is
              not part of your public profile and other residents cannot see it.
            </p>
          </div>
        </section>
      </main>

      <footer className="mx-auto max-w-5xl px-6 pb-12">
        <p className="border-t border-border pt-6 text-xs text-ink-faint">
          Nativity Guard — community safety, on the record.
        </p>
      </footer>
    </div>
  )
}
