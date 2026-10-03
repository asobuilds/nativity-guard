import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { LayoutDashboard, Shield, Users } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { UnitTabs } from '@/components/unit/UnitTabs'
import { UnitInboxPanel } from '@/components/unit/UnitInboxPanel'
import { ComplianceMatrix } from '@/components/unit/ComplianceMatrix'
import { api } from '@/lib/apiClient'
import { useUnitInbox } from '@/hooks/useUnitInbox'
import { useUnitCompliance } from '@/hooks/useUnitCompliance'
import type { SecurityUnit } from '@/types/api'

interface Membership {
  id: string
  unitId: string
  role: string
  status: string
  isHeadAdmin: boolean
  unit?: SecurityUnit
}

type TabId = 'inbox' | 'compliance'

export function UnitDashboardPage() {
  const [activeTab, setActiveTab] = useState<TabId>('inbox')

  // Find the caller's active unit_admin membership.
  const membershipsQuery = useQuery({
    queryKey: ['my-unit-memberships'],
    queryFn: () =>
      api.get<{ memberships: Membership[] }>('/units/my-memberships'),
    select: (d) => d.memberships,
    staleTime: 60_000,
  })

  const adminMembership = membershipsQuery.data?.find(
    (m) =>
      m.status === 'active' &&
      (m.role === 'unit_admin' || m.isHeadAdmin) &&
      m.unitId,
  )

  const unitId = adminMembership?.unitId

  // The unit details (name, logo). Uses the standard unit endpoint.
  const unitQuery = useQuery({
    queryKey: ['unit', unitId],
    queryFn: () => api.get<{ unit: SecurityUnit }>(`/units/${unitId}`),
    select: (d) => d.unit,
    enabled: Boolean(unitId),
  })

  const inbox = useUnitInbox(unitId)
  const compliance = useUnitCompliance(unitId)

  if (membershipsQuery.isLoading) {
    return (
      <div className="mx-auto max-w-6xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-24 w-full rounded-panel" />
      </div>
    )
  }

  if (!adminMembership || !unitId) {
    return (
      <div className="mx-auto max-w-3xl p-4 sm:p-6">
        <Card className="p-8 text-center">
          <LayoutDashboard className="mx-auto size-10 text-signal" aria-hidden />
          <h1 className="mt-4 text-xl font-semibold text-ink">No unit attached</h1>
          <p className="mx-auto mt-2 max-w-md text-sm text-ink-muted">
            Your account is not an active administrator of any unit. Ask a platform
            administrator to promote you, or open an election from your unit's page.
          </p>
          <Link
            to="/units"
            className="mt-4 inline-block text-sm text-signal hover:underline"
          >
            Browse units →
          </Link>
        </Card>
      </div>
    )
  }

  const unit = unitQuery.data
  const unitName = unit?.brandName || unit?.name || 'Your unit'

  const tabs = [
    { id: 'inbox', label: 'Inbox', badge: (inbox.data?.pendingCaseCount ?? 0) + (inbox.data?.sosAlertCount ?? 0) + (inbox.data?.applicationCount ?? 0) },
    { id: 'compliance', label: 'Weekly compliance' },
  ]

  return (
    <div className="mx-auto max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <Link
          to={`/units/${unitId}`}
          className="text-sm text-signal hover:underline"
        >
          ← {unitName}
        </Link>
        <h1 className="mt-2 text-2xl font-semibold text-ink">Unit dashboard</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Today's work for {unitName}. Cases, SOS, membership applications, and
          whether officers are filing their weekly updates.
        </p>
      </header>

      <UnitTabs tabs={tabs} activeId={activeTab} onChange={(id) => setActiveTab(id as TabId)} />

      {activeTab === 'inbox' ? (
        inbox.isError ? (
          <Card>
            <ErrorState
              title="Could not load the inbox"
              description="Try again in a moment."
              onRetry={() => void inbox.refetch()}
            />
          </Card>
        ) : (
          <UnitInboxPanel inbox={inbox.data} isLoading={inbox.isLoading} />
        )
      ) : null}

      {activeTab === 'compliance' ? (
        <Card className="p-5">
          <div className="flex items-start gap-3">
            <Users className="mt-0.5 size-5 shrink-0 text-signal" aria-hidden />
            <div>
              <h2 className="text-base font-semibold text-ink">Weekly compliance</h2>
              <p className="mt-1 text-sm text-ink-muted">
                Did each officer file their weekly case updates? A red cross is a
                week with nothing filed.
              </p>
            </div>
          </div>
          <div className="mt-4">
            {compliance.isError ? (
              <ErrorState
                title="Could not load compliance"
                description="Try again in a moment."
                onRetry={() => void compliance.refetch()}
              />
            ) : (
              <ComplianceMatrix
                compliance={compliance.data}
                isLoading={compliance.isLoading}
              />
            )}
          </div>
        </Card>
      ) : null}

      <Card className="border-signal/20 bg-signal/5 p-4">
        <div className="flex items-start gap-3">
          <Shield className="mt-0.5 size-4 shrink-0 text-signal" aria-hidden />
          <p className="text-xs text-ink-muted">
            Every action you take here is logged. Officers see what you assign and
            what you approve.
          </p>
        </div>
      </Card>
    </div>
  )
}
