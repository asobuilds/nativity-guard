import { useMemo, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { Handshake, Plus, Users } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { Badge } from '@/components/ui/Chips'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Field, Input, Select } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useCreateTaskForce, useMyTaskForces } from '@/hooks/useTaskForce'
import { useUnits } from '@/hooks/useUnits'
import { api, ApiError } from '@/lib/apiClient'
import type { TaskForceSummary } from '@/types/api'

type Membership = { unitId: string; role: string; status: string; isHeadAdmin: boolean }

function statusTone(status: string): 'ok' | 'warn' | 'neutral' {
  if (status === 'active') return 'ok'
  if (status === 'invited') return 'warn'
  return 'neutral'
}

function summaryLabel(summary: TaskForceSummary): string {
  if (summary.role === 'host') return 'Host unit'
  if (summary.membershipStatus === 'invited') return 'Invitation pending'
  return 'Member unit'
}

export function TaskForceListPage() {
  const { role } = useAuth()
  const navigate = useNavigate()
  const units = useUnits()
  const taskForces = useMyTaskForces()
  const memberships = useQuery({
    queryKey: ['my-unit-memberships'],
    queryFn: () => api.get<{ memberships: Membership[] }>('/units/my-memberships'),
    enabled: role !== 'super_admin',
  })
  const create = useCreateTaskForce()
  const [name, setName] = useState('')
  const [hostUnitId, setHostUnitId] = useState('')

  const headAdminUnits = useMemo(() => {
    if (role === 'super_admin') return units.data ?? []
    const ids = new Set(
      (memberships.data?.memberships ?? [])
        .filter((item) => item.status === 'active' && item.isHeadAdmin)
        .map((item) => item.unitId),
    )
    return (units.data ?? []).filter((unit) => ids.has(unit.id))
  }, [memberships.data, role, units.data])

  const selectedHost = headAdminUnits.some((unit) => unit.id === hostUnitId)
    ? hostUnitId
    : headAdminUnits[0]?.id ?? ''

  function submit(event: FormEvent) {
    event.preventDefault()
    if (!name.trim() || !selectedHost) return
    create.mutate(
      { name: name.trim(), hostUnitId: selectedHost },
      {
        onSuccess: (data) => navigate(`/taskforce/${data.taskForce.id}`),
      },
    )
  }

  const loading = units.isLoading || taskForces.isLoading || (role !== 'super_admin' && memberships.isLoading)

  return (
    <main className="mx-auto w-full max-w-5xl space-y-6 p-4 sm:p-6">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-widest text-signal">Multi-unit operations</p>
        <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
          <Handshake className="size-6 text-signal" aria-hidden /> Joint task forces
        </h1>
        <p className="max-w-2xl text-sm text-ink-muted">
          Coordinate officers from multiple units in one shared operations channel. Each officer may belong to only one active task force.
        </p>
      </header>

      <Card className="p-5">
        <h2 className="text-sm font-semibold text-ink">Create a task force</h2>
        {headAdminUnits.length === 0 ? (
          <p className="mt-2 text-sm text-ink-muted">
            Only the head admin of a verified unit can create a task force. Ask your unit head admin to start one.
          </p>
        ) : (
          <form className="mt-4 grid gap-4 sm:grid-cols-[1fr_1fr_auto] sm:items-end" onSubmit={submit}>
            <Field label="Task force name" htmlFor="taskforce-name">
              {(props) => (
                <Input
                  {...props}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="Joint border patrol"
                  maxLength={120}
                />
              )}
            </Field>
            <Field label="Host unit" htmlFor="taskforce-host">
              {(props) => (
                <Select {...props} value={selectedHost} onChange={(event) => setHostUnitId(event.target.value)}>
                  {headAdminUnits.map((unit) => (
                    <option key={unit.id} value={unit.id}>{unit.name}</option>
                  ))}
                </Select>
              )}
            </Field>
            <Button type="submit" variant="primary" icon={<Plus className="size-4" />} loading={create.isPending} disabled={!name.trim() || !selectedHost}>
              Create
            </Button>
          </form>
        )}
        {create.isError ? (
          <p role="alert" className="mt-3 text-sm text-emergency">
            {create.error instanceof Error ? create.error.message : 'Could not create the task force.'}
          </p>
        ) : null}
      </Card>

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {[0, 1].map((index) => <Skeleton key={index} className="h-40 rounded-panel" />)}
        </div>
      ) : units.isError || taskForces.isError ? (
        <Card>
          <ErrorState
            title="Could not load task forces"
            description="Try again to see your joint operations."
            offline={ApiError.isNetwork(units.error) || ApiError.isNetwork(taskForces.error)}
            onRetry={() => { void units.refetch(); void taskForces.refetch() }}
          />
        </Card>
      ) : (taskForces.data ?? []).length === 0 ? (
        <Card>
          <EmptyState
            icon={<Users className="size-6" />}
            title="No task forces yet"
            description={headAdminUnits.length ? 'Create one above, or wait for an invitation from another unit.' : 'You will see a task force here when your unit joins one.'}
          />
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {(taskForces.data ?? []).map((summary) => (
            <Card key={summary.taskForce.id} as="article" className="flex flex-col p-5 transition-colors hover:border-signal/60">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="truncate text-lg font-semibold text-ink">{summary.taskForce.name}</h2>
                  <p className="mt-1 text-sm text-ink-muted">{summary.taskForce.hostUnit?.name ?? 'Host unit'}</p>
                </div>
                <Badge tone={statusTone(summary.taskForce.status)}>{summary.taskForce.status}</Badge>
              </div>
              <p className="mt-4 text-xs text-ink-muted">{summaryLabel(summary)}</p>
              <Link
                to={`/taskforce/${summary.taskForce.id}`}
                className="mt-5 inline-flex items-center text-sm font-semibold text-signal underline-offset-4 hover:underline"
              >
                Open task force
              </Link>
            </Card>
          ))}
        </div>
      )}
    </main>
  )
}
