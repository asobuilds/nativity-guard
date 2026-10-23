import { useMemo, useState } from 'react'
import {
  Building2,
  Check,
  Copy,
  CreditCard,
  Search,
  ShieldAlert,
  ShieldCheck,
  UserPlus,
  Users,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Input, Select } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { useToast } from '@/components/ui/Toast'
import { usePublicUnits } from '@/hooks/usePublicUnits'
import { useGeoStates } from '@/hooks/useGeo'
import { api } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'
import type { ActionContext } from './types'

export interface UnitsDirectorySectionProps {
  onActionTrigger: (action: ActionContext) => void
}

interface UnitBankAccount {
  id: string
  bankName: string
  accountNumber: string
  accountName: string
  bankCode?: string
  branch?: string
  isDefault: boolean
}

export function UnitsDirectorySection({ onActionTrigger }: UnitsDirectorySectionProps) {
  const { data: units = [], isError: unitsError, isPending: unitsLoading } = usePublicUnits()
  const { data: states = [] } = useGeoStates()
  const { notify } = useToast()

  const [query, setQuery] = useState('')
  const [selectedState, setSelectedState] = useState('all')
  const [sortBy, setSortBy] = useState<'members' | 'name' | 'radius'>('members')

  // Direct Bank Donation Modal State
  const [activeDonationUnit, setActiveDonationUnit] = useState<SecurityUnit | null>(null)
  const [bankAccounts, setBankAccounts] = useState<UnitBankAccount[]>([])
  const [loadingAccounts, setLoadingAccounts] = useState(false)
  const [accountsError, setAccountsError] = useState(false)
  const [copiedKey, setCopiedKey] = useState<string | null>(null)

  const filteredUnits = useMemo(() => {
    const q = query.trim().toLowerCase()
    return units
      .filter((u) => {
        if (u.status !== 'active') return false
        if (selectedState !== 'all' && u.state?.toLowerCase() !== selectedState.toLowerCase()) {
          return false
        }
        if (!q) return true
        return (
          u.name.toLowerCase().includes(q) ||
          u.lga?.toLowerCase().includes(q) ||
          u.state?.toLowerCase().includes(q) ||
          u.registrationNumber?.toLowerCase().includes(q)
        )
      })
      .sort((a, b) => {
        if (sortBy === 'members') return (b.totalMembers ?? 0) - (a.totalMembers ?? 0)
        if (sortBy === 'radius') return b.operationalRadius - a.operationalRadius
        return a.name.localeCompare(b.name)
      })
  }, [units, query, selectedState, sortBy])

  async function handleOpenDonation(unit: SecurityUnit) {
    if (!unit.isVerified) return
    setActiveDonationUnit(unit)
    setLoadingAccounts(true)
    setAccountsError(false)
    setBankAccounts([])
    try {
      const res = await api.get<{ accounts: UnitBankAccount[] }>(
        `/public/units/${unit.id}/bank-accounts`
      )
      setBankAccounts(res.accounts || [])
    } catch {
      setBankAccounts([])
      setAccountsError(true)
    } finally {
      setLoadingAccounts(false)
    }
  }

  function handleCopy(text: string, key: string) {
    void navigator.clipboard.writeText(text)
    setCopiedKey(key)
    notify('Account number copied to clipboard', 'success')
    setTimeout(() => setCopiedKey(null), 2500)
  }

  function handleEnlist(unit: SecurityUnit) {
    onActionTrigger({
      type: 'enlist_officer',
      title: `Enlist with ${unit.name}`,
      description:
        'Official security officer duty requires background vetting, NIN validation, and local traditional ruler endorsement.',
      badgeText: `Unit: ${unit.name} (${unit.lga})`,
      returnUrl: `/units/${unit.id}`,
      payload: { unitId: unit.id, unitName: unit.name },
    })
  }

  function handleJoinCitizen(unit: SecurityUnit) {
    onActionTrigger({
      type: 'join_unit',
      title: `Join ${unit.name} as a Community Member`,
      description:
        'Become a verified community member to vote in annual leadership elections and participate in unit governance.',
      badgeText: `Unit: ${unit.name}`,
      returnUrl: `/units/${unit.id}`,
      payload: { unitId: unit.id },
    })
  }

  return (
    <section id="units-directory" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <Building2 className="size-3.5 text-signal" aria-hidden />
            Public security unit directory
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Community Security Units Directory
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            Browse units published by Nativity Guard. Verification is a platform status,
            not government approval. Check a unit's identity before engaging.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <Button
            variant="primary"
            size="sm"
            onClick={() =>
              onActionTrigger({
                type: 'join_unit',
                title: 'Register New Community Security Unit',
                description:
                  'Host a unit for your community. Units with 12+ verified members unlock democratic governance and public dispatch.',
                returnUrl: '/super/units/new',
              })
            }
            icon={<ShieldAlert className="size-3.5" />}
          >
            Register a Security Unit
          </Button>
        </div>
      </div>

      {/* Search and Filters */}
      <Card className="mt-6 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative min-w-56 flex-1">
            <Search
              className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
              aria-hidden
            />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search by unit name, LGA, or state..."
              className="pl-9 text-xs"
            />
          </div>

          <div className="w-48">
            <Select
              value={selectedState}
              onChange={(e) => setSelectedState(e.target.value)}
              className="text-xs py-1.5"
            >
              <option value="all">All States</option>
              {states.map((s) => (
                <option key={s.name} value={s.name}>
                  {s.name}
                </option>
              ))}
            </Select>
          </div>

          <div className="w-44">
            <Select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as 'members' | 'name' | 'radius')}
              className="text-xs py-1.5"
            >
              <option value="members">Sort: Most Members</option>
              <option value="radius">Sort: Largest Radius</option>
              <option value="name">Sort: Alphabetical</option>
            </Select>
          </div>
        </div>
      </Card>

      {unitsError ? (
        <p role="alert" className="mt-4 text-sm text-warn">Unable to load units from the service. No sample organisations are shown.</p>
      ) : unitsLoading ? (
        <p role="status" className="mt-4 text-sm text-ink-muted">Loading published units...</p>
      ) : filteredUnits.length === 0 ? (
        <p role="status" className="mt-4 text-sm text-ink-muted">No published active units match your filters.</p>
      ) : null}

      {/* Unit Cards Grid */}
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {filteredUnits.map((unit) => (
          <Card key={unit.id} className="flex flex-col justify-between p-5">
            <div>
              <div className="flex items-start justify-between gap-1">
                <span className="inline-flex items-center gap-1 rounded-full bg-ok/10 px-2 py-0.5 text-[10px] font-semibold text-ok ring-1 ring-ok/30">
                  <ShieldCheck className="size-3" aria-hidden />
                  {unit.isVerified ? 'Platform-verified' : 'Not verified'}
                </span>
                <span className="text-[10px] text-ink-faint tabular">
                  {unit.operationalRadius} km radius
                </span>
              </div>

              <h3 className="mt-3 text-sm font-semibold text-ink line-clamp-1 leading-snug">
                {unit.name}
              </h3>
              <p className="mt-1 text-[11px] text-ink-muted">
                {unit.lga}, {unit.state}
              </p>
              <p className="mt-0.5 text-[10px] text-ink-faint line-clamp-1">
                {unit.type}
              </p>

              <div className="mt-4 space-y-1.5 border-t border-border pt-3 text-[11px] text-ink-muted">
                <div className="flex items-center justify-between">
                  <span className="flex items-center gap-1 text-ink-faint">
                    <Users className="size-3" aria-hidden />
                    Roster:
                  </span>
                  <span className="font-medium text-ink tabular">{unit.totalMembers ?? '12+'} members</span>
                </div>
                {unit.registrationNumber ? (
                  <div className="flex items-center justify-between">
                    <span className="text-ink-faint">Reg No:</span>
                    <span className="font-mono text-[9px] text-ink-muted truncate max-w-28">
                      {unit.registrationNumber}
                    </span>
                  </div>
                ) : null}
              </div>
            </div>

            <div className="mt-5 space-y-2 border-t border-border pt-3">
              <Button
                variant="primary"
                size="sm"
                block
                className="text-xs"
                onClick={() => handleOpenDonation(unit)}
                disabled={!unit.isVerified}
                icon={<CreditCard className="size-3.5" />}
              >
                {unit.isVerified ? 'View donation details' : 'Donation unavailable'}
              </Button>

              <div className="grid grid-cols-2 gap-1.5">
                <Button
                  variant="secondary"
                  size="sm"
                  className="text-[11px] px-2"
                  onClick={() => handleJoinCitizen(unit)}
                >
                  Join Unit
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-[11px] px-2"
                  onClick={() => handleEnlist(unit)}
                  icon={<UserPlus className="size-3 text-signal" />}
                >
                  Enlist
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Direct Bank Donation Modal */}
      {activeDonationUnit ? (
        <Modal
          open={Boolean(activeDonationUnit)}
          onClose={() => setActiveDonationUnit(null)}
          title={`Support ${activeDonationUnit.name}`}
          size="md"
        >
          <div className="space-y-4">
            <div className="rounded-lg border border-ok/30 bg-ok/10 p-3.5 text-xs text-ink">
              <p className="font-semibold text-ok">Direct-to-Unit Financial Model</p>
              <p className="mt-1 text-ink-muted leading-relaxed">
                Any details below come from the platform's unit record. Confirm ownership directly
                with {activeDonationUnit.name} before making any transfer.
              </p>
            </div>

            {loadingAccounts ? (
              <p className="py-6 text-center text-xs text-ink-muted">Loading verified bank details...</p>
            ) : bankAccounts.length > 0 ? (
              <div className="space-y-3">
                {bankAccounts.map((account) => (
                  <div
                    key={account.id}
                    className="rounded-panel border border-border bg-surface-hi p-4 space-y-2.5"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-semibold text-signal uppercase tracking-wider">
                        {account.bankName}
                      </span>
                      {account.isDefault ? (
                        <span className="rounded bg-ok/20 px-1.5 py-0.5 text-[9px] font-bold text-ok">
                          Primary listed account
                        </span>
                      ) : null}
                    </div>

                    <div>
                      <p className="text-[11px] text-ink-faint">Account Name:</p>
                      <p className="font-medium text-xs text-ink">{account.accountName}</p>
                    </div>

                    <div className="flex items-center justify-between rounded-md bg-surface p-2.5 border border-border">
                      <div>
                        <p className="text-[10px] text-ink-faint">Account Number:</p>
                        <p className="font-mono text-sm font-bold text-signal tabular">
                          {account.accountNumber}
                        </p>
                      </div>

                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() =>
                          handleCopy(
                            account.accountNumber,
                            account.id
                          )
                        }
                        icon={
                          copiedKey === account.id ? (
                            <Check className="size-3.5 text-ok" />
                          ) : (
                            <Copy className="size-3.5" />
                          )
                        }
                      >
                        {copiedKey === account.id ? 'Copied' : 'Copy'}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="py-4 text-center text-xs text-ink-muted">
                {accountsError
                  ? 'Bank details could not be loaded. Do not transfer money using unverified details.'
                  : 'No bank account details are currently published by this unit.'}
              </p>
            )}

            <div className="border-t border-border pt-3 flex justify-end">
              <Button variant="secondary" size="sm" onClick={() => setActiveDonationUnit(null)}>
                Close
              </Button>
            </div>
          </div>
        </Modal>
      ) : null}
    </section>
  )
}
