import { useMemo, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { Copy, Handshake, LogOut, Radio, ShieldAlert, UserPlus, Users } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { BackLink } from '@/components/ui/BackLink'
import { Badge } from '@/components/ui/Chips'
import { Button } from '@/components/ui/Button'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Field, Input, Select } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { WalkieTalkie } from '@/components/unit/WalkieTalkie'
import {
  useDisbandTaskForce,
  useEnlistTaskForceOfficers,
  useInviteTaskForceUnit,
  useJoinTaskForce,
  useLeaveTaskForce,
  useTaskForce,
  useTaskForceOfficers,
} from '@/hooks/useTaskForce'
import { useUnits } from '@/hooks/useUnits'
import { api, ApiError } from '@/lib/apiClient'
import type { TaskForceMember } from '@/types/api'

type Membership = { unitId: string; role: string; status: string; isHeadAdmin: boolean }

function errorText(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback
}

function memberTone(status: string): 'ok' | 'warn' | 'neutral' | 'emergency' {
  if (status === 'active') return 'ok'
  if (status === 'invited') return 'warn'
  if (status === 'removed') return 'emergency'
  return 'neutral'
}

export function TaskForcePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user, role } = useAuth()
  const detail = useTaskForce(id)
  const units = useUnits()
  const memberships = useQuery({
    queryKey: ['my-unit-memberships'],
    queryFn: () => api.get<{ memberships: Membership[] }>('/units/my-memberships'),
    enabled: role !== 'super_admin',
  })

  const canManage = detail.data?.canManage ?? false
  const officers = useTaskForceOfficers(id, canManage)
  const invite = useInviteTaskForceUnit(id)
  const join = useJoinTaskForce(id)
  const enlist = useEnlistTaskForceOfficers(id)
  const disband = useDisbandTaskForce(id)
  const leave = useLeaveTaskForce(id)

  const [inviteUnitId, setInviteUnitId] = useState('')
  const [inviteHours, setInviteHours] = useState('24')
  const [inviteCode, setInviteCode] = useState('')
  const [joinCode, setJoinCode] = useState('')
  const [selectedOfficerIds, setSelectedOfficerIds] = useState<Set<string>>(new Set())
  const [confirmDisband, setConfirmDisband] = useState(false)
  const [actionMessage, setActionMessage] = useState('')

  const unitNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const unit of units.data ?? []) map.set(unit.id, unit.name)
    return map
  }, [units.data])

  const members = useMemo(() => detail.data?.members ?? [], [detail.data?.members])
  const activeTaskForceOfficerIds = useMemo(
    () => new Set((detail.data?.officers ?? []).map((item) => item.officerId)),
    [detail.data?.officers],
  )

  const availableInviteUnits = useMemo(() => {
    const joined = new Set(members.map((member) => member.unitId))
    return (units.data ?? []).filter((unit) => !joined.has(unit.id))
  }, [members, units.data])

  const selectedInviteUnit = availableInviteUnits.some((unit) => unit.id === inviteUnitId)
    ? inviteUnitId
    : availableInviteUnits[0]?.id ?? ''

  const availableOfficers = useMemo(
    () => (officers.data ?? []).filter((officer) => officer.status === 'active' && !activeTaskForceOfficerIds.has(officer.id)),
    [activeTaskForceOfficerIds, officers.data],
  )

  const myMembership = useMemo(() => {
    if (!user?.unitId) return undefined
    return (memberships.data?.memberships ?? []).find(
      (item) => item.unitId === user.unitId && item.status === 'active',
    )
  }, [memberships.data, user?.unitId])

  const myTaskForceMembership: TaskForceMember | undefined = useMemo(() => {
    if (!user?.unitId) return undefined
    return members.find((member) => member.unitId === user.unitId)
  }, [members, user?.unitId])

  const canJoin = Boolean(
    myMembership?.isHeadAdmin && myTaskForceMembership?.status === 'invited',
  )
  const canLeave = Boolean(
    myMembership?.isHeadAdmin &&
      myTaskForceMembership?.status === 'active' &&
      myTaskForceMembership?.role === 'member',
  )

  function submitInvite(event: FormEvent) {
    event.preventDefault()
    if (!selectedInviteUnit) return
    invite.mutate(
      { unitId: selectedInviteUnit, expiresInHours: Number(inviteHours) || undefined },
      {
        onSuccess: (data) => {
          setInviteCode(data.code)
          setActionMessage(`${unitNames.get(selectedInviteUnit) ?? 'Unit'} was invited. Share the code privately with that unit's head admin.`)
        },
      },
    )
  }

  function submitJoin(event: FormEvent) {
    event.preventDefault()
    if (!joinCode.trim()) return
    join.mutate(
      { code: joinCode.trim() },
      { onSuccess: () => { setJoinCode(''); setActionMessage('Your unit joined the task force.') } },
    )
  }

  function toggleOfficer(officerId: string) {
    setSelectedOfficerIds((previous) => {
      const next = new Set(previous)
      if (next.has(officerId)) next.delete(officerId)
      else next.add(officerId)
      return next
    })
  }

  function submitEnlist() {
    if (selectedOfficerIds.size === 0) return
    enlist.mutate([...selectedOfficerIds], {
      onSuccess: (data) => {
        setSelectedOfficerIds(new Set())
        setActionMessage(`${data.enlisted} officer${data.enlisted === 1 ? '' : 's'} enlisted.`)
      },
    })
  }

  if (detail.isLoading) {
    return (
      <main className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-64 rounded-panel" />
      </main>
    )
  }

  if (detail.isError || !detail.data) {
    return (
      <main className="mx-auto w-full max-w-3xl p-4 sm:p-6">
        <BackLink to="/taskforce" label="Back to task forces" className="mb-4" />
        <Card>
          <ErrorState
            title="Task force unavailable"
            description={errorText(detail.error, 'It may have been disbanded, or your unit may not be a member.')}
            offline={ApiError.isNetwork(detail.error)}
            onRetry={() => void detail.refetch()}
          />
        </Card>
      </main>
    )
  }

  const { taskForce, channel, online, ptt } = detail.data
  const isActive = taskForce.status === 'active'

  return (
    <main className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6">
      <BackLink to="/taskforce" label="Back to task forces" />

      <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
              <Handshake className="size-6 text-signal" aria-hidden /> {taskForce.name}
            </h1>
            <Badge tone={isActive ? 'ok' : 'neutral'}>{taskForce.status}</Badge>
          </div>
          <p className="text-sm text-ink-muted">
            Hosted by {taskForce.hostUnit?.name ?? unitNames.get(taskForce.hostUnitId) ?? 'the host unit'}
          </p>
        </div>
        {canManage && isActive ? (
          confirmDisband ? (
            <div className="flex items-center gap-2">
              <Button variant="danger" size="sm" loading={disband.isPending} onClick={() => disband.mutate(undefined, { onSuccess: () => navigate('/taskforce') })}>
                Confirm disband
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setConfirmDisband(false)}>Cancel</Button>
            </div>
          ) : (
            <Button variant="secondary" size="sm" icon={<ShieldAlert className="size-4" />} onClick={() => setConfirmDisband(true)}>
              Disband
            </Button>
          )
        ) : null}
      </header>

      {actionMessage ? (
        <div role="status" className="rounded-lg border border-ok/30 bg-ok/10 px-3 py-2 text-sm text-ok">
          {actionMessage}
        </div>
      ) : null}

      {inviteCode ? (
        <Card className="border-warn/40 p-4">
          <p className="text-sm font-semibold text-ink">Invite code</p>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <code className="rounded-lg bg-surface-hi px-3 py-2 font-mono text-sm tracking-wider text-ink">{inviteCode}</code>
            <Button
              size="sm"
              variant="secondary"
              icon={<Copy className="size-4" />}
              onClick={() => void navigator.clipboard?.writeText(inviteCode)}
            >
              Copy
            </Button>
          </div>
          <p className="mt-2 text-xs text-warn">Share this privately with the invited unit's head admin. It is shown only once.</p>
        </Card>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)]">
        <div className="space-y-6">
          <Card>
            <CardHeader title="Units" subtitle={`${members.filter((member) => member.status === 'active').length} active member unit(s)`} />
            <CardBody className="space-y-3">
              {members.length === 0 ? (
                <p className="text-sm text-ink-muted">No units have joined yet.</p>
              ) : (
                members.map((member) => (
                  <div key={member.id} className="flex items-center justify-between gap-3 rounded-lg border border-border bg-surface-hi/50 px-3 py-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-ink">{unitNames.get(member.unitId) ?? member.unit?.name ?? 'Unit'}</p>
                      <p className="text-xs text-ink-muted">{member.role === 'host' ? 'Host unit' : 'Member unit'}</p>
                    </div>
                    <Badge tone={memberTone(member.status)}>{member.status}</Badge>
                  </div>
                ))
              )}

              {canJoin ? (
                <form className="space-y-2 border-t border-border pt-3" onSubmit={submitJoin}>
                  <Field label="Join your unit" htmlFor="taskforce-join-code" hint="Enter the invite code sent to your unit head admin.">
                    {(props) => (
                      <Input
                        {...props}
                        value={joinCode}
                        onChange={(event) => setJoinCode(event.target.value)}
                        placeholder="Invite code"
                        autoComplete="off"
                      />
                    )}
                  </Field>
                  <Button type="submit" size="sm" variant="primary" loading={join.isPending} disabled={!joinCode.trim()}>
                    Join task force
                  </Button>
                  {join.isError ? <p role="alert" className="text-xs text-emergency">{errorText(join.error, 'Could not join.')}</p> : null}
                </form>
              ) : null}

              {canLeave ? (
                <div className="flex items-center justify-between gap-3 border-t border-border pt-3">
                  <p className="text-xs text-ink-muted">Leaving removes your unit and releases its enlisted officers.</p>
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={<LogOut className="size-4" />}
                    loading={leave.isPending}
                    onClick={() => leave.mutate(user?.unitId ?? undefined, { onSuccess: () => navigate('/taskforce') })}
                  >
                    Leave
                  </Button>
                </div>
              ) : null}
            </CardBody>
          </Card>

          <Card>
            <CardHeader title="Enlisted officers" subtitle={`${detail.data.officers.length} officer(s)`} />
            <CardBody className="space-y-3">
              {detail.data.officers.length === 0 ? (
                <EmptyState icon={<Users className="size-5" />} title="No officers enlisted" description="Head admins can enlist active officers from member units." />
              ) : (
                detail.data.officers.map((item) => (
                  <div key={item.id} className="flex items-center justify-between gap-3 rounded-lg border border-border bg-surface-hi/50 px-3 py-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium text-ink">{item.officer?.name ?? item.officerId}</p>
                      <p className="text-xs text-ink-muted">
                        {[item.officer?.rank, item.officer?.badgeNumber, unitNames.get(item.unitId) ?? item.unit?.name].filter(Boolean).join(' · ')}
                      </p>
                    </div>
                    <Badge tone={item.status === 'active' ? 'ok' : 'neutral'}>{item.status}</Badge>
                  </div>
                ))
              )}
            </CardBody>
          </Card>
        </div>

        <div className="space-y-6">
          {isActive ? (
            <Card>
              <CardHeader
                title={<span className="flex items-center gap-2"><Radio className="size-4 text-signal" aria-hidden /> Task force channel</span>}
                subtitle={`${online} online · ${ptt.speakerName ? `${ptt.speakerName} transmitting` : 'Channel open'}`}
              />
              <div className="p-4">
                {channel ? (
                  <WalkieTalkie channel={{ kind: 'taskforce', id: taskForce.id }} channelName={taskForce.name} />
                ) : (
                  <p className="text-sm text-ink-muted">The task force channel is not available yet.</p>
                )}
              </div>
            </Card>
          ) : null}

          {canManage && isActive ? (
            <>
              <Card>
                <CardHeader title="Invite a unit" subtitle="Only unit head admins can invite another unit." />
                <CardBody className="space-y-3">
                  {availableInviteUnits.length === 0 ? (
                    <p className="text-sm text-ink-muted">Every unit is already attached to this task force.</p>
                  ) : (
                    <form className="space-y-3" onSubmit={submitInvite}>
                      <Field label="Unit" htmlFor="taskforce-invite-unit">
                        {(props) => (
                          <Select {...props} value={selectedInviteUnit} onChange={(event) => setInviteUnitId(event.target.value)}>
                            {availableInviteUnits.map((unit) => <option key={unit.id} value={unit.id}>{unit.name}</option>)}
                          </Select>
                        )}
                      </Field>
                      <Field label="Expires in hours" htmlFor="taskforce-invite-hours">
                        {(props) => (
                          <Input {...props} type="number" min={1} max={168} value={inviteHours} onChange={(event) => setInviteHours(event.target.value)} />
                        )}
                      </Field>
                      <Button type="submit" variant="primary" icon={<UserPlus className="size-4" />} loading={invite.isPending} disabled={!selectedInviteUnit}>
                        Generate invite
                      </Button>
                      {invite.isError ? <p role="alert" className="text-xs text-emergency">{errorText(invite.error, 'Could not create invite.')}</p> : null}
                    </form>
                  )}
                </CardBody>
              </Card>

              <Card>
                <CardHeader title="Enlist officers" subtitle="Active officers from member units only." />
                <CardBody className="space-y-3">
                  {officers.isLoading ? (
                    <Skeleton className="h-24" />
                  ) : availableOfficers.length === 0 ? (
                    <p className="text-sm text-ink-muted">No available officers. Officers already enlisted in another active task force must be removed there first.</p>
                  ) : (
                    <>
                      <div className="max-h-72 space-y-2 overflow-y-auto pr-1">
                        {availableOfficers.map((officer) => (
                          <label key={officer.id} className="flex cursor-pointer items-start gap-3 rounded-lg border border-border bg-surface-hi/50 px-3 py-2">
                            <input
                              type="checkbox"
                              className="mt-1 size-4 accent-signal"
                              checked={selectedOfficerIds.has(officer.id)}
                              onChange={() => toggleOfficer(officer.id)}
                            />
                            <span className="min-w-0">
                              <span className="block truncate text-sm font-medium text-ink">{officer.name}</span>
                              <span className="block text-xs text-ink-muted">
                                {[officer.rank, officer.badgeNumber, unitNames.get(officer.unitId)].filter(Boolean).join(' · ')}
                              </span>
                            </span>
                          </label>
                        ))}
                      </div>
                      <Button variant="primary" onClick={submitEnlist} loading={enlist.isPending} disabled={selectedOfficerIds.size === 0}>
                        Enlist selected ({selectedOfficerIds.size})
                      </Button>
                      {enlist.isError ? <p role="alert" className="text-xs text-emergency">{errorText(enlist.error, 'Could not enlist officers.')}</p> : null}
                      {enlist.data && enlist.data.results.some((result) => result.status !== 'enlisted') ? (
                        <div className="space-y-1 text-xs text-warn">
                          {enlist.data.results.filter((result) => result.status !== 'enlisted').map((result, index) => (
                            <p key={`${result.officerId ?? index}`}>{result.officerId ?? 'Officer'}: {result.message ?? result.status}</p>
                          ))}
                        </div>
                      ) : null}
                    </>
                  )}
                </CardBody>
              </Card>
            </>
          ) : null}

          {!canManage && !canJoin && !canLeave ? (
            <Card>
              <CardBody>
                <p className="text-sm text-ink-muted">
                  Task force setup is managed by the host unit's head admin. {role === 'super_admin' ? 'You have full oversight access.' : 'You can listen on the shared channel while your unit is an active member.'}
                </p>
              </CardBody>
            </Card>
          ) : null}
        </div>
      </div>
    </main>
  )
}
