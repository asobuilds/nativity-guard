import { Link } from 'react-router-dom'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { ArrowRight, Mail, MessageSquare, Shield, Users } from 'lucide-react'

/**
 * `/about` — public. Explains what Nativity Guard is, what it is not, who runs
 * it, and how to reach support. No auth required.
 *
 * The wording is deliberately conservative: nothing here claims the platform
 * does something it does not do, and the emergency disclaimer at the top is
 * the first thing a reader sees.
 */
export function AboutPage() {
  return (
    <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <Link to="/" className="text-sm text-signal hover:underline">
          ← Nativity Guard
        </Link>
        <h1 className="mt-3 text-3xl font-bold tracking-tight text-ink">About Nativity Guard</h1>
        <p className="mt-3 text-base text-ink-muted">
          A coordination platform that connects citizens with independent community security
          units on one shared record.
        </p>
      </header>

      <Card className="border border-warn/30 bg-warn/5">
        <CardBody>
          <div className="flex items-start gap-3">
            <Shield className="mt-0.5 size-5 shrink-0 text-warn" aria-hidden />
            <div>
              <p className="text-sm font-medium text-ink">
                Not an emergency service
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                If you need immediate help, call your local emergency number. A report or SOS sent
                through this platform may be delayed, may not be received, and does not guarantee
                a response.
              </p>
            </div>
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="What Nativity Guard is" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <p>
            Nativity Guard is a coordination platform. It connects residents with independent
            community security units operating in their area. A report filed here reaches the unit
            the reporter chose, appears in that unit's queue, and stays followable from filing to
            closure — every status change is on the record.
          </p>
          <p>
            It is not a security provider, an emergency service, a police force, or a dispatch
            agency.
          </p>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="How it works in practice" />
        <CardBody>
          <ol className="space-y-3 text-sm text-ink-muted">
            <li className="flex gap-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-signal/20 text-[11px] font-semibold text-signal">
                1
              </span>
              <span>
                <strong className="text-ink">You file a report.</strong> A short description, a
                pin on the map, the unit you want to handle it. You get a tracking ID immediately.
              </span>
            </li>
            <li className="flex gap-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-signal/20 text-[11px] font-semibold text-signal">
                2
              </span>
              <span>
                <strong className="text-ink">The unit triages it.</strong> An officer is assigned
                and reviews it. You get a notification when the status changes.
              </span>
            </li>
            <li className="flex gap-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-signal/20 text-[11px] font-semibold text-signal">
                3
              </span>
              <span>
                <strong className="text-ink">Closure is approved.</strong> The officer who worked
                the case cannot mark it finished. An administrator approves it or sends it back
                with a reason. Two people, always.
              </span>
            </li>
          </ol>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Developer non-interference" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <p>Nativity Guard, its developers, operators and affiliates:</p>
          <ul className="ml-5 list-disc space-y-1.5">
            <li>
              Do not direct, supervise, train, employ, or control any security unit that responds
              to activity on this platform.
            </li>
            <li>
              Do not investigate, verify, or guarantee the accuracy, completeness, or legality of
              any case report, SOS alert, or other content submitted by any user.
            </li>
            <li>
              Do not influence, approve, or take part in any tactical, operational, or
              investigative decision made by any security unit.
            </li>
            <li>
              Do not dispatch, direct, or coordinate the physical response of any security unit.
            </li>
          </ul>
          <p>
            All operational decisions are the sole responsibility of the responding unit and are
            made independently of the platform.
          </p>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Who runs Nativity Guard" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <p>
            The platform is operated by the Nativity Guard team. Data controller contact details
            will be published here once legal review is complete.
          </p>
          <p>
            Read the full{' '}
            <Link to="/terms" className="text-signal hover:underline">
              Terms of Service and Privacy Notice
            </Link>
            .
          </p>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Support and feedback" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <p>
            If you have a question, found a bug, or want to suggest an improvement, the fastest
            way to reach the team is through the in-app feedback form.
          </p>
          <div className="flex flex-col gap-2 pt-1">
            <Link
              to="/feedback"
              className="inline-flex items-center gap-2 text-signal hover:underline"
            >
              <MessageSquare className="size-4" aria-hidden />
              Send feedback or report a problem
              <ArrowRight className="size-3.5" aria-hidden />
            </Link>
            <a
              href="mailto:support@nativityguard.com"
              className="inline-flex items-center gap-2 text-signal hover:underline"
            >
              <Mail className="size-4" aria-hidden />
              support@nativityguard.com
            </a>
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Coverage" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <div className="flex items-start gap-3">
            <Users className="mt-0.5 size-5 shrink-0 text-signal" aria-hidden />
            <p>
              Community security units are registered and verified individually. Availability
              varies by area — open the{' '}
              <Link to="/map" className="text-signal hover:underline">
                safety map
              </Link>{' '}
              to see which units operate near you.
            </p>
          </div>
        </CardBody>
      </Card>

      <footer className="pt-4 text-center text-xs text-ink-faint">
        Nativity Guard — community safety, on the record.
      </footer>
    </main>
  )
}