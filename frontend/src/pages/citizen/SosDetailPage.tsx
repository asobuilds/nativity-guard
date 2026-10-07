import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { AlertTriangle, CheckCircle2, MapPin, ShieldAlert, Users } from 'lucide-react'
import { Badge, PriorityChip } from '@/components/ui/Chips'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Modal } from '@/components/ui/Modal'
import { Select } from '@/components/ui/Field'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { SosLiveTracking } from '@/components/sos/SosLiveTracking'
import { SosTracker } from '@/components/sos/SosTracker'
import { useAuth } from '@/auth/AuthContext'
import { useAcceptSos, useAssignSos, useReleaseSos, useSosDetail } from '@/hooks/useSos'
import { useUnits } from '@/hooks/useUnits'
import { isSosActive } from '@/lib/sosTracking'
import { formatDateTime, relativeTime } from '@/lib/format'

function dispatchLabel(state: string | undefined): string {
  switch (state) {
    case 'open': return 'Open — waiting for a unit'
    case 'locked': return 'Locked — one unit responding'
    case 'multi_responder': return 'Multiple units responding'
    case 'resolved': return 'Resolved'
    default: return 'Unknown'
  }
}

export function SosDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { user, role } = useAuth()
  const detail = useSosDetail(id)
  const accept = useAcceptSos()
  const release = useReleaseSos()
  const assign = useAssignSos()
  const units = useUnits()

  const [assignOpen, setAssignOpen] = useState(false)
  const [assignUnitId, setAssignUnitId] = useState('')

  const sos = detail.data
  const responders = sos?.responders ?? []
  const isReporter = user?.id === sos?.userId
  const isUnitAdmin = role === 'unit_admin'
  const isSuperAdmin = role === 'super_admin'
  const myUnitId = user?.unitId ?? null
  const iAmResponding = myUnitId ? responders.some((r) => r.unitId === myUnitId) : false
  const isCritical = sos?.severity === 'critical'
  const hasCapacity = isCritical ? responders.length < 3 : responders.length === 0
  const canAccept = isUnitAdmin && !iAmResponding && hasCapacity && sos?.dispatchState !== 'resolved'

  if (detail.isLoading) {
    return <div className="mx-auto max-w-4xl p-4 sm:p-6"><Skeleton className="h-64 w-full" /></div>
  }
  if (detail.isError || !sos) {
    return (
      <div className="mx-auto max-w-4xl p-4 sm:p-6">
        <Card><ErrorState title="SOS unavailable" description="This alert could not be found." onRetry={() => void detail.refetch()} /></Card>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-4 p-4 sm:p-6">
      <Link to="/sos" className="text-sm text-signal">← Back to SOS</Link>

      <Card className="space-y-3 p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <ShieldAlert className="size-5 text-emergency" aria-hidden />
              <h1 className="text-xl font-semibold text-ink">SOS alert</h1>
              {isCritical ? <Badge tone="warn">Critical</Badge> : null}
            </div>
            <p className="mt-1 text-sm text-ink-muted">Reported {relativeTime(sos.createdAt)}</p>
          </div>
          <PriorityChip level={sos.priority} />
        </div>

        <p className="rounded-lg border border-border bg-surface-hi/50 p-3 text-sm text-ink">
          {dispatchLabel(sos.dispatchState)}
        </p>

        {sos.description ? (
          <div>
            <p className="text-xs uppercase tracking-widest text-ink-faint">Description</p>
            <p className="mt-1 text-sm text-ink-muted">{sos.description}</p>
          </div>
        ) : null}

        {sos.jurisdictionState || sos.jurisdictionLga ? (
          <p className="flex items-center gap-1.5 text-sm text-ink-muted">
            <MapPin className="size-4" aria-hidden />
            {[sos.jurisdictionLga, sos.jurisdictionState].filter(Boolean).join(', ')}
          </p>
        ) : null}

        {isReporter ? (
          <div className="rounded-lg border border-ok/40 bg-ok/10 p-3 text-sm text-ink">
            <CheckCircle2 className="inline size-4 text-ok" aria-hidden /> Your alert is visible to responders. Follow its progress below.
          </div>
        ) : null}
      </Card>

      <Card className="space-y-3 p-5">
        <h2 className="text-sm font-semibold text-ink">Progress</h2>
        <SosTracker alert={sos} />
      </Card>

      <SosLiveTracking sos={sos} />

      <Card className="space-y-4 p-5">
        <div className="flex items-center gap-2">
          <Users className="size-4 text-signal" aria-hidden />
          <h2 className="text-sm font-semibold text-ink">Responders ({responders.length}{isCritical ? '/3' : ''})</h2>
        </div>

        {responders.length === 0 ? (
          <p className="text-sm text-ink-muted">No unit has accepted yet.</p>
        ) : (
          <ul className="space-y-2">
            {responders.map((r) => (
              <li key={r.id} className="flex items-center justify-between rounded-lg border border-border p-3">
                <div>
                  <p className="text-sm font-medium text-ink">{r.unit?.name ?? 'Unit'}</p>
                  <p className="mt-0.5 text-xs text-ink-faint">{r.role} · accepted {relativeTime(r.acceptedAt)}</p>
                </div>
                <Badge tone={r.role === 'primary' ? 'ok' : undefined}>{r.role}</Badge>
              </li>
            ))}
          </ul>
        )}

        <div className="flex flex-wrap gap-2 border-t border-border pt-4">
          {canAccept ? (
            <Button
              variant="primary"
              icon={<AlertTriangle className="size-4" />}
              loading={accept.isPending}
              onClick={() => id && accept.mutate(id)}
            >
              Accept SOS
            </Button>
          ) : null}

          {isUnitAdmin && iAmResponding && isSosActive(sos) ? (
            <Button variant="ghost" loading={release.isPending} onClick={() => id && release.mutate(id)}>
              Release
            </Button>
          ) : null}

          {isSuperAdmin ? (
            <Button variant="secondary" onClick={() => setAssignOpen(true)}>
              Assign to unit
            </Button>
          ) : null}

          {isUnitAdmin && !canAccept && !iAmResponding ? (
            <p className="text-xs text-ink-faint">This SOS is not open to your unit.</p>
          ) : null}
        </div>

        {accept.isError ? <p role="alert" className="text-xs text-warn">Accept failed. It may already be taken.</p> : null}
        {release.isError ? <p role="alert" className="text-xs text-warn">Release failed.</p> : null}
      </Card>

      <p className="text-xs text-ink-faint">Created {formatDateTime(sos.createdAt)}</p>

      {assignOpen ? (
        <Modal open onClose={() => setAssignOpen(false)} title="Assign to a security unit">
          <p className="text-sm text-ink-muted">Force-assign this SOS. This bypasses the normal lock.</p>
          <div className="mt-4">
            <Select value={assignUnitId} onChange={(e) => setAssignUnitId(e.target.value)}>
              <option value="">Choose a unit…</option>
              {(units.data ?? []).map((u) => (
                <option key={u.id} value={u.id}>{u.name}</option>
              ))}
            </Select>
          </div>
          {assign.isError ? <p className="mt-3 text-sm text-warn">Assign failed.</p> : null}
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setAssignOpen(false)} disabled={assign.isPending}>Cancel</Button>
            <Button
              loading={assign.isPending}
              disabled={!assignUnitId}
              onClick={() => {
                if (!id || !assignUnitId) return
                assign.mutate({ sosId: id, unitId: assignUnitId }, {
                  onSuccess: () => { setAssignOpen(false); setAssignUnitId('') },
                })
              }}
            >
              Assign
            </Button>
          </div>
        </Modal>
      ) : null}
    </div>
  )
}