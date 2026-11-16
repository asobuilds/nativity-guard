import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { Field, Input, Select } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { Badge } from '@/components/ui/Chips'
import { useGeoStates, useGeoLgas } from '@/hooks/useGeo'
import { useUnits } from '@/hooks/useUnits'
import { getEligibleAgencyUnits } from '@/lib/agency'
import type { AgencyType } from '@/hooks/useAgencyTenants'
import { useAgencyTenants, useCreateAgencyTenant, useAssignAgencyAdmin, useLinkAgencyTenantUnit } from '@/hooks/useAgencyTenants'

const AGENCY_LABEL: Record<AgencyType, string> = {
  police: 'Nigeria Police Force',
  nscdc: 'NSCDC',
  fire: 'Federal Fire Service',
  frsc: 'FRSC',
}

export function SuperAgencyManagementPage() {
  const tenants = useAgencyTenants()
  const createTenant = useCreateAgencyTenant()
  const assignAdmin = useAssignAgencyAdmin()
  const linkUnit = useLinkAgencyTenantUnit()
  const units = useUnits()
  const states = useGeoStates()

  const [createOpen, setCreateOpen] = useState(false)
  const [assignOpen, setAssignOpen] = useState<string | null>(null)
  const [linkOpen, setLinkOpen] = useState<string | null>(null)
  const [selectedUnitId, setSelectedUnitId] = useState('')
  const [agencyType, setAgencyType] = useState<AgencyType>('police')
  const [state, setState] = useState('')
  const [lga, setLga] = useState('')
  const [divisionName, setDivisionName] = useState('')
  const [email, setEmail] = useState('')
  const [rank, setRank] = useState('')
  const lgas = useGeoLgas(state)
  const linkingTenant = tenants.data?.tenants.find((tenant) => tenant.id === linkOpen)
  const eligibleUnits = getEligibleAgencyUnits(
    units.data ?? [],
    linkingTenant,
    tenants.data?.tenants.flatMap((tenant) => tenant.unitId ? [tenant.unitId] : []) ?? [],
  )

  function submitCreate() {
    createTenant.mutate(
      { agencyType, state, lga, divisionName },
      {
        onSuccess: () => {
          setCreateOpen(false)
          setAgencyType('police')
          setState('')
          setLga('')
          setDivisionName('')
        },
      }
    )
  }

  function submitLink() {
    if (!linkOpen || !selectedUnitId) return
    linkUnit.mutate(
      { tenantId: linkOpen, unitId: selectedUnitId },
      { onSuccess: () => { setLinkOpen(null); setSelectedUnitId('') } },
    )
  }

  function submitAssign() {
    if (!assignOpen) return
    assignAdmin.mutate(
      { tenantId: assignOpen, email, rank },
      {
        onSuccess: () => {
          setAssignOpen(null)
          setEmail('')
          setRank('')
        },
      }
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-4 p-4 sm:p-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-ink">Agency divisions</h1>
          <p className="mt-1 text-sm text-ink-muted">
            {tenants.data?.tenants.length ?? 0} registered oversight divisions
          </p>
        </div>
        <Button variant="primary" onClick={() => setCreateOpen(true)}>
          <Plus className="size-4" /> Create tenant
        </Button>
      </header>

      {tenants.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : tenants.isError ? (
        <ErrorState title="Could not load agency tenants" description="Try again." onRetry={() => tenants.refetch()} />
      ) : tenants.data?.tenants.length ? (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[46rem] text-left text-sm">
            <thead className="border-b border-border text-xs uppercase tracking-wider text-ink-muted">
              <tr>
                <th className="px-4 py-3">Division</th>
                <th className="px-4 py-3">Agency</th>
                <th className="px-4 py-3">Jurisdiction</th>
                <th className="px-4 py-3">Division Admin</th>
                <th className="px-4 py-3">Operational Unit</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tenants.data.tenants.map((t) => (
                <tr key={t.id} className="border-b border-border/60 last:border-0">
                  <td className="px-4 py-3 font-medium text-ink">{t.divisionName}</td>
                  <td className="px-4 py-3"><Badge>{AGENCY_LABEL[t.agencyType]}</Badge></td>
                  <td className="px-4 py-3 text-ink-muted">{t.lga}, {t.state}</td>
                  <td className="px-4 py-3 text-ink-muted">
                    {t.admin ? (
                      <div>
                        <div className="font-medium text-ink">{t.admin.firstName} {t.admin.lastName}</div>
                        <div className="text-xs">{t.admin.email}</div>
                      </div>
                    ) : (
                      <Badge>Unassigned</Badge>
                    )}
                  </td>
                  <td className="px-4 py-3 text-ink-muted">
                    {t.unitId ? (
                      <Badge>Linked · {units.data?.find((unit) => unit.id === t.unitId)?.name ?? 'Operational unit'}</Badge>
                    ) : (
                      <Badge>Not linked</Badge>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" variant="secondary" onClick={() => setAssignOpen(t.id)}>
                        Assign DPO
                      </Button>
                      {!t.unitId ? (
                        <Button size="sm" variant="secondary" onClick={() => { setLinkOpen(t.id); setSelectedUnitId('') }}>
                          Link unit
                        </Button>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      ) : (
        <EmptyState
          title="No agency divisions yet"
          description="Create a division to grant oversight access."
          action={<Button variant="primary" onClick={() => setCreateOpen(true)}>Create tenant</Button>}
        />
      )}

      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="Create agency division" description="This creates an oversight tenant for a specific agency and jurisdiction.">
        <div className="space-y-3">
          <Field label="Agency">
            {() => (
              <Select value={agencyType} onChange={(e) => setAgencyType(e.target.value as AgencyType)}>
                <option value="police">Nigeria Police Force</option>
                <option value="nscdc">NSCDC</option>
                <option value="fire">Federal Fire Service</option>
                <option value="frsc">FRSC</option>
              </Select>
            )}
          </Field>
          <Field label="State">
            {() => (
              <Select value={state} onChange={(e) => { setState(e.target.value); setLga('') }}>
                <option value="">Select a state</option>
                {states.data?.map((s) => <option key={s.name} value={s.name}>{s.name}</option>)}
              </Select>
            )}
          </Field>
          <Field label="LGA">
            {() => (
              <Select value={lga} onChange={(e) => setLga(e.target.value)} disabled={!state}>
                <option value="">Select an LGA</option>
                {lgas.data?.map((l) => <option key={l} value={l}>{l}</option>)}
              </Select>
            )}
          </Field>
          <Field label="Division name">
            {() => <Input value={divisionName} onChange={(e) => setDivisionName(e.target.value)} placeholder="e.g. Ikeja Police Division" />}
          </Field>
          {createTenant.isError && <p className="text-sm text-emergency">{(createTenant.error as Error)?.message}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setCreateOpen(false)}>Cancel</Button>
            <Button variant="primary" disabled={!state || !lga || !divisionName} loading={createTenant.isPending} onClick={submitCreate}>Create</Button>
          </div>
        </div>
      </Modal>


      <Modal open={!!linkOpen} onClose={() => { setLinkOpen(null); setSelectedUnitId('') }} title="Link operational unit" description={linkingTenant ? `Choose an active, verified ${AGENCY_LABEL[linkingTenant.agencyType]} unit in ${linkingTenant.lga}, ${linkingTenant.state}. A unit can belong to only one agency division.` : 'Choose an eligible operational unit.'}>
        <div className="space-y-3">
          <Field label="Verified operational unit">
            {() => (
              <Select value={selectedUnitId} onChange={(e) => setSelectedUnitId(e.target.value)} disabled={units.isLoading || eligibleUnits.length === 0}>
                <option value="">Select an operational unit</option>
                {eligibleUnits.map((unit) => <option key={unit.id} value={unit.id}>{unit.name} · {unit.type} · {unit.lga}, {unit.state}</option>)}
              </Select>
            )}
          </Field>
          {units.isError ? <ErrorState title="Could not load operational units" description="Refresh and try again." onRetry={() => void units.refetch()} /> : null}
          {!units.isLoading && !units.isError && eligibleUnits.length === 0 ? <p className="text-sm text-ink-muted">No unlinked, active and verified units match this agency type and jurisdiction. Verify or register the correct unit first.</p> : null}
          {linkUnit.isError ? <p className="text-sm text-emergency" role="alert">{(linkUnit.error as Error).message}</p> : null}
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => { setLinkOpen(null); setSelectedUnitId('') }}>Cancel</Button>
            <Button variant="primary" disabled={!selectedUnitId} loading={linkUnit.isPending} onClick={submitLink}>Link unit</Button>
          </div>
        </div>
      </Modal>

      <Modal open={!!assignOpen} onClose={() => setAssignOpen(null)} title="Assign division admin" description="Enter the email of a registered user who will oversee this division.">
        <div className="space-y-3">
          <Field label="User email">
            {() => <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="dpo@police.gov.ng" />}
          </Field>
          <Field label="Rank (optional)">
            {() => <Input value={rank} onChange={(e) => setRank(e.target.value)} placeholder="e.g. DPO" />}
          </Field>
          {assignAdmin.isError && <p className="text-sm text-emergency">{(assignAdmin.error as Error)?.message}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setAssignOpen(null)}>Cancel</Button>
            <Button variant="primary" disabled={!email} loading={assignAdmin.isPending} onClick={submitAssign}>Assign</Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
