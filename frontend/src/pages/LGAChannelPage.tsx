import { Link } from 'react-router-dom'
import { ArrowLeft, Building2 } from 'lucide-react'
import { LGAChannelPanel } from '@/components/unit/LGAChannelPanel'

/**
 * `/lga` — the private inter-unit channel for the caller's LGA.
 *
 * Access is decided by the backend: a verified member of a verified unit
 * whose unit carries an LGA. Citizens and unverified units get a 403 from
 * the endpoint, and this page renders the panel's own error state.
 */
export function LGAChannelPage() {
  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
      <Link
        to="/"
        className="inline-flex items-center gap-1.5 text-sm text-signal hover:underline"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        Back to home
      </Link>

      <header className="flex items-center gap-3">
        <div className="grid size-11 place-items-center rounded-full bg-signal/15 ring-1 ring-signal/30">
          <Building2 className="size-5 text-signal" aria-hidden />
        </div>
        <div>
          <h1 className="text-2xl font-semibold text-ink">LGA channel</h1>
          <p className="text-sm text-ink-muted">
            Private bulletin board for verified units in your local government area.
          </p>
        </div>
      </header>

      <LGAChannelPanel />
    </div>
  )
}
