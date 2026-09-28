import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Search, ShieldAlert, ShieldCheck, UserCog } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { Input, Select } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { useAllUsers, useSuspendUser, useActivateUser, useUpdateUserRole, type SuperAdminUser } from '@/hooks/useSuperAdmin'
import { useAuth } from '@/auth/AuthContext'
import { relativeTime } from '@/lib/format'
import type { Role } from '@/types/api'

const PAGE_SIZE = 20
const ROLES: Role[] = ['citizen', 'officer', 'unit_admin', 'super_admin']
const ROLE_LABEL: Record<Role, string> = {
  citizen: 'Citizen', officer: 'Officer', unit_admin: 'Unit admin', super_admin: 'Super admin',
}

export function SuperUsersPage() {
  const { user: me } = useAuth()
  const { data, isLoading, isError, refetch } = useAllUsers()
  const suspend = useSuspendUser()
  const activate = useActivateUser()
  const updateRole = useUpdateUserRole()

  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<'all' | Role>('all')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'pending' | 'suspended'>('all')
  const [page, setPage] = useState(1)
  const [roleModal, setRoleModal] = useState<SuperAdminUser | null>(null)
  const [roleDraft, setRoleDraft] = useState<Role>('citizen')

  useEffect(() => { setPage(1) }, [search, roleFilter, statusFilter])

  const filtered = useMemo(() => {
    const list = data?.users ?? []
    const q = search.trim().toLowerCase()
    return list.filter((u) => {
      if (roleFilter !== 'all' && u.role !== roleFilter) return false
      if (statusFilter !== 'all' && u.status !== statusFilter) return false
      if (!q) return true
      const hay = [u.firstName, u.lastName, u.email, u.phone].filter(Boolean).join(' ').toLowerCase()
      return hay.includes(q)
    })
  }, [data?.users, search, roleFilter, statusFilter])

  const pageCount = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const paginated = useMemo(() => filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), [filtered, page])

  function openRoleModal(u: SuperAdminUser) {
    setRoleModal(u)
    setRoleDraft(u.role)
  }

  function saveRole() {
    if (!roleModal) return
    updateRole.mutate({ userId: roleModal.id, role: roleDraft }, {
      onSuccess: () => setRoleModal(null),
    })
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Users</h1>
        <p className="mt-1 text-sm text-ink-muted">{data?.total ?? 0} accounts on the platform.</p>
      </header>

      <Card>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
            <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search name, email, phone" className="pl-9" />
          </div>
          <Select value={roleFilter} onChange={(e) => setRoleFilter(e.target.value as typeof roleFilter)}>
            <option value="all">All roles</option>
            {ROLES.map((r) => <option key={r} value={r}>{ROLE_LABEL[r]}</option>)}
          </Select>
          <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}>
            <option value="all">All statuses</option>
            <option value="active">Active</option>
            <option value="pending">Pending</option>
            <option value="suspended">Suspended</option>
          </Select>
        </div>
      </Card>

      {isLoading ? (
        <Card><Skeleton className="h-64 w-full" /></Card>
      ) : isError ? (
        <Card><ErrorState title="Could not load users" description="Try again." onRetry={() => void refetch()} /></Card>
      ) : filtered.length === 0 ? (
        <Card><EmptyState title="No users match" description="Adjust the filters above." /></Card>
      ) : (
        <>
          <Card className="overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left text-xs uppercase tracking-wider text-ink-faint">
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3">Email</th>
                    <th className="px-4 py-3">Role</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Joined</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {paginated.map((u) => {
                    const isSelf = u.id === me?.id
                    const fullName = [u.firstName, u.lastName].filter(Boolean).join(' ') || u.email
                    return (
                      <tr key={u.id} className="border-b border-border last:border-0 hover:bg-surface-hi">
                        <td className="px-4 py-3">
                          {isSelf ? <Link to="/profile" className="text-signal hover:underline">{fullName}</Link> : <span className="text-ink">{fullName}</span>}
                        </td>
                        <td className="px-4 py-3 text-ink-muted">{u.email}</td>
                        <td className="px-4 py-3"><Badge>{ROLE_LABEL[u.role] ?? u.role}</Badge></td>
                        <td className="px-4 py-3">
                          <Badge tone={u.status === 'suspended' ? 'warn' : u.status === 'active' ? 'ok' : undefined}>{u.status}</Badge>
                        </td>
                        <td className="px-4 py-3 text-ink-muted">{relativeTime(u.createdAt)}</td>
                        <td className="px-4 py-3">
                          <div className="flex justify-end gap-2">
                            {u.status === 'active' && !isSelf && (
                              <Button size="sm" variant="ghost" icon={<ShieldAlert className="size-3.5" />} onClick={() => { if (window.confirm('Suspend ' + fullName + '?')) suspend.mutate(u.id) }} disabled={suspend.isPending}>Suspend</Button>
                            )}
                            {u.status === 'suspended' && (
                              <Button size="sm" variant="ghost" icon={<ShieldCheck className="size-3.5" />} onClick={() => activate.mutate(u.id)} disabled={activate.isPending}>Activate</Button>
                            )}
                            {!isSelf && (
                              <Button size="sm" variant="ghost" icon={<UserCog className="size-3.5" />} onClick={() => openRoleModal(u)}>Role</Button>
                            )}
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </Card>

          <div className="flex items-center justify-between text-sm">
            <span className="text-ink-muted">Page {page} of {pageCount} · {filtered.length} users</span>
            <div className="flex gap-2">
              <Button size="sm" variant="ghost" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>Previous</Button>
              <Button size="sm" variant="ghost" onClick={() => setPage((p) => Math.min(pageCount, p + 1))} disabled={page >= pageCount}>Next</Button>
            </div>
          </div>
        </>
      )}

      {roleModal && (
        <Modal open onClose={() => setRoleModal(null)} title="Change role">
          <p className="text-sm text-ink-muted">Set the role for <strong className="text-ink">{roleModal.firstName} {roleModal.lastName}</strong>.</p>
          <div className="mt-4">
            <Select value={roleDraft} onChange={(e) => setRoleDraft(e.target.value as Role)}>
              {ROLES.map((r) => <option key={r} value={r}>{ROLE_LABEL[r]}</option>)}
            </Select>
          </div>
          {updateRole.isError && <p className="mt-3 text-sm text-warn">Could not update role. Try again.</p>}
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setRoleModal(null)} disabled={updateRole.isPending}>Cancel</Button>
            <Button onClick={saveRole} loading={updateRole.isPending} disabled={roleDraft === roleModal.role}>Save</Button>
          </div>
        </Modal>
      )}
    </div>
  )
}