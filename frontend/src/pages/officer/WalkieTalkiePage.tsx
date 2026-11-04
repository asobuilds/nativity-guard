import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Radio } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { useUnits } from '@/hooks/useUnits'
import { Card } from '@/components/ui/Card'
import { Select } from '@/components/ui/Field'
import { WalkieTalkie } from '@/components/unit/WalkieTalkie'

/**
 * Full-screen unit channel. Officers and unit admins are pinned to their own
 * unit; super admins may pick any unit to listen in, with transmit gated behind
 * the command-override switch.
 */
export function WalkieTalkiePage() {
  const { user, role } = useAuth()
  const [params] = useSearchParams()
  const unitsQuery = useUnits()
  const isSuperAdmin = role === 'super_admin'
  const [selectedUnit, setSelectedUnit] = useState(
    () => params.get('unitId') || user?.unitId || '',
  )

  const unitId = isSuperAdmin ? selectedUnit : (user?.unitId ?? '')
  const unitName = useMemo(() => {
    if (!unitId) return undefined
    return unitsQuery.data?.find((unit) => unit.id === unitId)?.name
  }, [unitId, unitsQuery.data])

  return (
    <div className="mx-auto w-full max-w-4xl p-4 sm:p-6">
      <header className="mb-4">
        <h1 className="flex items-center gap-2 text-xl font-semibold text-ink">
          <Radio className="size-5 text-signal" aria-hidden />
          Walkie-talkie
        </h1>
        <p className="mt-1 text-sm text-ink-muted">
          Push-to-talk with your unit. One officer transmits at a time and every
          transmission releases automatically after 60 seconds.
        </p>
      </header>

      {isSuperAdmin ? (
        <div className="mb-4 max-w-sm">
          <label htmlFor="walkie-unit" className="mb-1.5 block text-xs font-medium text-ink-muted">
            Unit channel
          </label>
          <Select
            id="walkie-unit"
            value={selectedUnit}
            onChange={(event) => setSelectedUnit(event.target.value)}
          >
            <option value="">Select a unit…</option>
            {(unitsQuery.data ?? []).map((unit) => (
              <option key={unit.id} value={unit.id}>
                {unit.name}
              </option>
            ))}
          </Select>
          <p className="mt-1.5 text-[11px] text-ink-faint">
            Listening is always allowed. Enable command override in the panel to transmit.
          </p>
        </div>
      ) : null}

      {unitId ? (
        <WalkieTalkie channel={{ kind: 'unit', id: unitId }} channelName={unitName} />
      ) : (
        <Card className="p-6 text-center text-sm text-ink-muted">
          {isSuperAdmin
            ? 'Select a unit channel to listen in.'
            : 'You are not assigned to a unit, so there is no channel to join.'}
        </Card>
      )}
    </div>
  )
}
