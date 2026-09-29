import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Mail, MapPin, Phone, Shield, Users } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { MapView } from '@/components/map/MapView'
import { api } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'

export function UnitDetailPage() {
  const { id } = useParams<{ id: string }>()
  const query = useQuery({
    queryKey: ['unit', id],
    queryFn: () => api.get<{ unit: SecurityUnit }>(`/units/${id}`),
    select: (d) => d.unit,
    enabled: Boolean(id),
  })

  if (query.isLoading) {
    return <div className="mx-auto max-w-4xl p-4 sm:p-6"><Skeleton className="h-64 w-full" /></div>
  }
  if (query.isError || !query.data) {
    return <div className="mx-auto max-w-4xl p-4 sm:p-6"><Card><ErrorState title="Unit unavailable" description="This unit could not be found." onRetry={() => void query.refetch()} /></Card></div>
  }

  const unit = query.data
  const hasCoords = Number.isFinite(unit.latitude) && Number.isFinite(unit.longitude) && unit.latitude !== 0 && unit.longitude !== 0

  return (
    <div className="mx-auto max-w-4xl space-y-4 p-4 sm:p-6">
      <Link to="/units" className="text-sm text-signal">← Back to units</Link>

      <Card className="p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold text-ink">{unit.name}</h1>
            <p className="mt-1 text-sm text-ink-muted">{unit.type}</p>
          </div>
          {unit.verificationStatus === 'verified' ? (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-signal/10 px-3 py-1 text-xs text-signal">
              <Shield className="size-3.5" aria-hidden /> Verified
            </span>
          ) : null}
        </div>
        {[unit.city, unit.lga, unit.state, unit.ward].filter(Boolean).length > 0 ? (
          <p className="mt-3 flex items-center gap-1.5 text-sm text-ink-muted">
            <MapPin className="size-4" aria-hidden />
            {[unit.city, unit.lga, unit.state, unit.ward].filter(Boolean).join(', ')}
          </p>
        ) : null}
      </Card>

      {hasCoords ? (
        <Card className="overflow-hidden">
          <MapView
            mode="view"
            cases={[]}
            units={[unit]}
            showUnitCoverage
            showHotspots={false}
            height="40vh"
            allowLocate={false}
          />
        </Card>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2">
        {unit.contactPhone ? (
          <Card className="flex items-center gap-3 p-4">
            <Phone className="size-4 text-signal" aria-hidden />
            <div>
              <p className="text-xs text-ink-faint">Phone</p>
              <a href={`tel:${unit.contactPhone}`} className="text-sm text-ink hover:underline">{unit.contactPhone}</a>
            </div>
          </Card>
        ) : null}
        {unit.contactEmail ? (
          <Card className="flex items-center gap-3 p-4">
            <Mail className="size-4 text-signal" aria-hidden />
            <div>
              <p className="text-xs text-ink-faint">Email</p>
              <a href={`mailto:${unit.contactEmail}`} className="text-sm text-ink hover:underline">{unit.contactEmail}</a>
            </div>
          </Card>
        ) : null}
        <Card className="flex items-center gap-3 p-4">
          <Users className="size-4 text-signal" aria-hidden />
          <div>
            <p className="text-xs text-ink-faint">Members</p>
            <p className="text-sm text-ink">{unit.totalMembers ?? 0}</p>
          </div>
        </Card>
        <Card className="flex items-center gap-3 p-4">
          <MapPin className="size-4 text-signal" aria-hidden />
          <div>
            <p className="text-xs text-ink-faint">Coverage radius</p>
            <p className="text-sm text-ink">{unit.operationalRadius ?? 10} km</p>
          </div>
        </Card>
      </div>

      {unit.coverageArea ? (
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-ink">Coverage area</h2>
          <p className="mt-2 whitespace-pre-wrap text-sm text-ink-muted">{unit.coverageArea}</p>
        </Card>
      ) : null}

      {unit.commanderName ? (
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-ink">Commander</h2>
          <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
            <div><dt className="text-ink-faint">Name</dt><dd className="text-ink">{unit.commanderName}</dd></div>
            {unit.commanderPhoneAlt ? <div><dt className="text-ink-faint">Alt phone</dt><dd className="text-ink">{unit.commanderPhoneAlt}</dd></div> : null}
            {unit.commanderOccupation ? <div><dt className="text-ink-faint">Occupation</dt><dd className="text-ink">{unit.commanderOccupation}</dd></div> : null}
          </dl>
        </Card>
      ) : null}

      <div className="flex gap-2">
        <Link to="/report"><Button variant="primary">Report near this unit</Button></Link>
      </div>
    </div>
  )
}