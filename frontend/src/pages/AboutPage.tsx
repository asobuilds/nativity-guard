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
                For urgent help in Nigeria, call 112. A report or SOS sent through this platform
                may be delayed, may not be received, and does not guarantee a response.
              </p>
            </div>
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="What Nativity Guard is" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <p>
            Nativity Guard is designed to help residents submit reports to participating community
            security units and follow available status updates. Actual service coverage, receipt,
            follow-up, and response depend on the participating unit and platform availability.
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
                pin on the map, and an available receiving unit. A tracking ID is issued if the
                submission succeeds.
              </span>
            </li>
            <li className="flex gap-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-signal/20 text-[11px] font-semibold text-signal">
                2
              </span>
              <span>
                <strong className="text-ink">A unit may review it.</strong> Depending on local
                participation and availability, officers may provide case progress updates.
              </span>
            </li>
            <li className="flex gap-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-signal/20 text-[11px] font-semibold text-signal">
                3
              </span>
              <span>
                <strong className="text-ink">Review and closure.</strong> Participating units
                may apply administrative review rules before marking a case closed. Do not assume
                that opening a report triggers a guaranteed review or physical response.
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
            Nativity Guard is the name used by this platform. A registered legal-entity name,
            formal business address, and named data-controller contact have not yet been
            independently verified for publication. Until those details are published, this page
            should not be treated as proof of registration, accreditation, or regulatory compliance.
          </p>
          <p>
            Read the full{' '}
            <Link to="/terms#terms" className="text-signal hover:underline">
              Terms of Service
            </Link>{' '}and{' '}
            <Link to="/terms#privacy" className="text-signal hover:underline">
              Privacy Notice and data rights information
            </Link>
            .
          </p>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Support and feedback" />
        <CardBody className="space-y-3 text-sm text-ink-muted">
          <p>
            The in-app feedback form requires sign-in. The support email below is published as a
            contact channel, but successful mailbox delivery has not been independently checked.
          </p>
          <div className="flex flex-col gap-2 pt-1">
            <Link
              to="/feedback"
              className="inline-flex items-center gap-2 text-signal hover:underline"
            >
              <MessageSquare className="size-4" aria-hidden />
              Sign in to send feedback or report a problem
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
              Availability of participating units varies by area. Platform verification is not
              government approval. Open the{' '}
              <Link to="/#map" className="text-signal hover:underline">
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