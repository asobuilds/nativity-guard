import { useMemo, useState } from 'react'
import {
  ArrowRight,
  CheckCircle2,
  ChevronDown,
  Clock,
  FileSearch,
  Lock,
  Plus,
  RotateCcw,
  Search,
  UserCheck,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Field'
import { PriorityChip, StatusChip } from '@/components/ui/Chips'
import { CASE_STATUS_META, FIELD_LIFECYCLE, REVIEW_PHASE } from '@/lib/status'
import { usePublicCases } from '@/hooks/usePublicCases'
import { relativeTime } from '@/lib/format'
import type { CaseStatus } from '@/types/api'
import type { ActionContext, PublicCase } from './types'

export interface CaseLifecycleSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

export function CaseLifecycleSection({ onActionTrigger }: CaseLifecycleSectionProps) {
  const { data: cases = [], isError: casesError } = usePublicCases()

  const [searchQuery, setSearchQuery] = useState('')
  const [selectedStatus, setSelectedStatus] = useState<CaseStatus | 'all'>('all')

  const searchResult = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return null
    return (
      cases.find(
        (c) =>
          c.trackingId.toLowerCase() === q ||
          c.id.toLowerCase() === q ||
          c.title.toLowerCase().includes(q)
      ) ?? null
    )
  }, [cases, searchQuery])

  const filteredCases = useMemo(() => {
    return cases.filter((c) => {
      if (selectedStatus !== 'all' && c.status !== selectedStatus) return false
      return true
    })
  }, [cases, selectedStatus])

  function handleFollowCase(caseItem: PublicCase) {
    onActionTrigger({
      type: 'follow_case',
      title: `Track Case ${caseItem.trackingId}`,
      description: `Sign in with your verified citizen account to follow case updates, submit counter-statements, or rate responders.`,
      badgeText: `Tracking Reference: ${caseItem.trackingId}`,
      returnUrl: `/cases/${caseItem.id}`,
      payload: { caseId: caseItem.id, trackingId: caseItem.trackingId },
    })
  }

  function handleReportNew() {
    onActionTrigger({
      type: 'report_incident',
      title: 'File an Official Incident Report',
      description: 'Describe what you saw, attach media evidence, and dispatch to your local security unit queue.',
      returnUrl: '/report',
    })
  }

  return (
    <section id="case-lifecycle" className="border-t border-border py-14">
      <div>
        <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
          <FileSearch className="size-3.5 text-signal" aria-hidden />
          Case Updates
        </span>
        <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
          Track a Case. Stay Informed.
        </h2>
        <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
          Use your reference number to see available public updates. Private case information requires sign-in. You can also explore the review stages below.
        </p>
      </div>

      {/* Tracking ID Lookup Tool */}
      <Card className="mt-8 p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h3 className="text-base font-semibold text-ink">Find a Report</h3>
            <p className="mt-0.5 text-xs text-ink-muted">
              Enter the tracking reference from your own report to check public updates.
            </p>
          </div>

          <div className="flex items-center gap-2 max-w-md w-full sm:w-auto">
            <div className="relative flex-1">
              <Search
                className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
                aria-hidden
              />
              <Input
                type="text"
                aria-label="Search by case reference"
                placeholder="Enter case reference"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 font-mono text-xs"
              />
            </div>
            {searchQuery ? (
              <Button variant="ghost" size="sm" onClick={() => setSearchQuery('')}>
                Clear
              </Button>
            ) : null}
          </div>
        </div>

        {/* Search Results Display */}
        {searchQuery.trim() ? (
          <div className="mt-4 border-t border-border pt-4">
            {casesError ? (
              <p role="alert" className="rounded-lg border border-warn/30 p-4 text-xs text-warn">
                Public case lookup is unavailable. Please try again later; no sample case results are shown.
              </p>
            ) : searchResult ? (
              <div className="rounded-lg border border-signal/30 bg-signal/5 p-4">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs font-semibold text-signal">
                      {searchResult.trackingId}
                    </span>
                    <StatusChip status={searchResult.status} />
                    <PriorityChip level={searchResult.priorityLevel} />
                  </div>
                  <span className="text-xs text-ink-faint">
                    Updated {relativeTime(searchResult.updatedAt)}
                  </span>
                </div>

                <h4 className="mt-2 text-sm font-medium text-ink">{searchResult.title}</h4>
                <p className="mt-1 text-xs text-ink-muted">{searchResult.description}</p>
                <p className="mt-2 text-[11px] text-ink-faint">
                  Jurisdiction Area: {searchResult.location}
                </p>

                <div className="mt-3 flex items-center justify-end">
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => handleFollowCase(searchResult)}
                    icon={<Lock className="size-3.5" />}
                  >
                    Authenticate to Follow Detailed Case Dossier
                  </Button>
                </div>
              </div>
            ) : (
              <div className="rounded-lg border border-border bg-surface-hi/40 p-4 text-center">
                <p className="text-xs font-medium text-ink">No public record matched "{searchQuery}"</p>
                <p className="mt-1 text-[11px] text-ink-muted">
                  Cases with privacy restrictions, confidential evidence, or unassigned citizen reports
                  are shielded. If you reported this case, sign in with your account to access it.
                </p>
                <div className="mt-3 flex justify-center">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() =>
                      onActionTrigger({
                        type: 'follow_case',
                        title: 'Locate Private Case Record',
                        description: 'Sign in to access your confidential citizen reports.',
                        badgeText: `Reference: ${searchQuery}`,
                      })
                    }
                  >
                    Sign In to Look Up Private Reports
                  </Button>
                </div>
              </div>
            )}
          </div>
        ) : null}
      </Card>

      {casesError ? (
        <p role="alert" className="mt-4 text-xs text-warn">Public case records could not be loaded from the service.</p>
      ) : null}

      {/* Full workflow is available on demand; tracking is the primary task. */}
      <details className="group mt-6 rounded-xl border border-border bg-surface/60">
        <summary className="flex cursor-pointer list-none items-center justify-between gap-3 p-4 text-sm font-semibold text-ink hover:bg-surface-hi/40 [&::-webkit-details-marker]:hidden">
          View the full case review process
          <ChevronDown className="size-4 shrink-0 text-signal transition-transform group-open:rotate-180" aria-hidden />
        </summary>
        <div className="border-t border-border px-4 pb-4">
      {/* 8-Stage Interactive Lifecycle Visualizer */}
      <div className="mt-8">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
          1. Field Operations (Sequential)
        </h3>
        <ol className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          {FIELD_LIFECYCLE.map((status, index) => {
            const meta = CASE_STATUS_META[status]
            const isSelected = selectedStatus === status
            return (
              <li
                key={status}
                onClick={() => setSelectedStatus(isSelected ? 'all' : status)}
                className={`cursor-pointer rounded-panel border p-4 transition-all ${
                  isSelected
                    ? 'border-signal bg-signal/10 shadow-panel'
                    : 'border-border bg-surface hover:border-border-hi'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="grid size-5 place-items-center rounded-full bg-surface-hi text-[11px] font-semibold text-ink-muted tabular">
                    {index + 1}
                  </span>
                  <span
                    className={`size-2 rounded-full ${meta.dot}`}
                    aria-hidden
                  />
                </div>
                <p className="mt-2 text-sm font-medium text-ink">{meta.label}</p>
                <p className="mt-1 text-xs leading-relaxed text-ink-muted">{meta.description}</p>
              </li>
            )
          })}
        </ol>

        <h3 className="mt-6 text-xs font-semibold uppercase tracking-wider text-ink-muted">
          2. Administrative Review & Closure Loop (Two-Admin Rule)
        </h3>
        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          {REVIEW_PHASE.map((status) => {
            const meta = CASE_STATUS_META[status]
            const isSelected = selectedStatus === status
            const isChanges = status === 'admin_changes_requested'
            const isClosed = status === 'closed'
            return (
              <div
                key={status}
                onClick={() => setSelectedStatus(isSelected ? 'all' : status)}
                className={`cursor-pointer rounded-panel border p-4 transition-all ${
                  isSelected
                    ? 'border-signal bg-signal/10 shadow-panel'
                    : 'border-border bg-surface hover:border-border-hi'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span
                    className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ring-1 ${meta.bg} ${meta.text} ${meta.ring}`}
                  >
                    {meta.label}
                  </span>
                  {isChanges ? (
                    <RotateCcw className="size-4 text-status-changes" aria-hidden />
                  ) : isClosed ? (
                    <CheckCircle2 className="size-4 text-ok" aria-hidden />
                  ) : (
                    <UserCheck className="size-4 text-signal" aria-hidden />
                  )}
                </div>
                <p className="mt-2 text-xs leading-relaxed text-ink-muted">{meta.description}</p>
                {isClosed ? (
                  <p className="mt-2 text-[10px] text-ok font-medium">
                    ✓ Requires 2 Admin Approvals
                  </p>
                ) : isChanges ? (
                  <p className="mt-2 text-[10px] text-status-changes font-medium">
                    ↺ Loops back to investigation
                  </p>
                ) : null}
              </div>
            )
          })}
        </div>
      </div>

        </div>
      </details>

      {/* Live Public Cases Stream */}
      <div className="mt-8">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-ink">
            Public Incident Updates{' '}
            {selectedStatus !== 'all' ? (
              <span className="text-xs font-normal text-ink-faint">
                (Filtered by {selectedStatus})
              </span>
            ) : null}
          </h3>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleReportNew}
            icon={<Plus className="size-3.5 text-signal" />}
          >
            File New Incident
          </Button>
        </div>

        <div className="mt-4 grid gap-3 md:grid-cols-3">
          {filteredCases.slice(0, 3).map((item) => (
            <Card key={item.id} className="flex flex-col justify-between p-4">
              <div>
                <div className="flex items-center justify-between gap-1">
                  <StatusChip status={item.status} />
                  <span className="font-mono text-[10px] text-ink-faint">{item.trackingId}</span>
                </div>
                <h4 className="mt-2.5 text-xs font-semibold text-ink line-clamp-1">
                  {item.title}
                </h4>
                <p className="mt-1 text-[11px] leading-relaxed text-ink-muted line-clamp-2">
                  {item.description}
                </p>
              </div>

              <div className="mt-4 border-t border-border pt-2.5">
                <div className="flex items-center justify-between text-[10px] text-ink-faint">
                  <span className="flex items-center gap-1">
                    <Clock className="size-3" aria-hidden />
                    {relativeTime(item.createdAt)}
                  </span>
                  <span>{item.location?.split(',')[0]}</span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  block
                  className="mt-2 text-xs"
                  onClick={() => handleFollowCase(item)}
                >
                  <span>Track Case</span>
                  <ArrowRight className="size-3 text-signal" aria-hidden />
                </Button>
              </div>
            </Card>
          ))}
        </div>
      </div>
    </section>
  )
}
