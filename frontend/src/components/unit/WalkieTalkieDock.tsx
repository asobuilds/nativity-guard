import { useState } from 'react'
import { Radio } from 'lucide-react'
import { WalkieTalkie } from './WalkieTalkie'

/**
 * Floating unit walkie-talkie for the officer console. Stays out of the way
 * until opened so it never competes with the SOS button or mobile nav.
 */
export function WalkieTalkieDock({ unitId, unitName }: { unitId: string | null; unitName?: string }) {
  const [open, setOpen] = useState(false)
  if (!unitId) return null

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="fixed bottom-20 left-4 z-30 inline-flex min-h-11 items-center gap-2 rounded-full border border-border-hi bg-surface px-4 text-sm font-semibold text-ink shadow-panel transition-colors hover:bg-surface-hi lg:bottom-5"
        aria-label="Open unit walkie-talkie"
      >
        <span className="relative grid size-5 place-items-center">
          <Radio className="size-4 text-signal" aria-hidden />
          <span className="absolute -right-1 -top-1 size-2 rounded-full bg-signal" aria-hidden />
        </span>
        <span className="hidden sm:inline">Walkie-talkie</span>
      </button>
    )
  }

  return (
    <div className="fixed bottom-20 left-4 z-30 w-[min(92vw,22rem)] lg:bottom-5">
      <WalkieTalkie
        unitId={unitId}
        unitName={unitName}
        variant="dock"
        onClose={() => setOpen(false)}
      />
    </div>
  )
}
