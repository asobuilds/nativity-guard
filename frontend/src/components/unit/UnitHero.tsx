import { Shield, ShieldCheck } from 'lucide-react'
import { mediaURL } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'

interface UnitHeroProps {
  unit: SecurityUnit
}

export function UnitHero({ unit }: UnitHeroProps) {
  const coverUrl = mediaURL(unit.brandCoverUrl) ?? null
  const logoUrl = mediaURL(unit.brandLogoUrl) ?? null
  const displayName = unit.brandName || unit.name
  const verified = unit.verificationStatus === 'verified'

  return (
    <div className="overflow-hidden rounded-panel border border-border bg-surface shadow-panel">
      <div className="relative h-48 w-full overflow-hidden bg-gradient-to-r from-signal/25 via-signal/10 to-warn/20 sm:h-56">
        {coverUrl ? (
          <img src={coverUrl} alt="" className="h-full w-full object-cover" />
        ) : null}
      </div>

      <div className="relative px-6 pb-6">
        <div className="-mt-16 sm:-mt-20">
          {logoUrl ? (
            <img
              src={logoUrl}
              alt={`${displayName} logo`}
              className="size-32 rounded-2xl border-4 border-base bg-base object-cover shadow-panel sm:size-40"
            />
          ) : (
            <div className="grid size-32 place-items-center rounded-2xl border-4 border-base bg-surface-hi text-signal shadow-panel sm:size-40">
              <Shield className="size-14 sm:size-16" aria-hidden />
            </div>
          )}
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-ink sm:text-4xl">
            {displayName}
          </h1>
          {verified ? (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-signal/15 px-3 py-1 text-xs font-medium text-signal ring-1 ring-signal/30">
              <ShieldCheck className="size-3.5" aria-hidden />
              Verified
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-xs font-medium text-ink-muted ring-1 ring-border">
              Verification pending
            </span>
          )}
        </div>

        <p className="mt-1 text-base text-ink-muted">{unit.type}</p>
      </div>
    </div>
  )
}
