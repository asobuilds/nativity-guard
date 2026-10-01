import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  BellRing,
  CalendarRange,
  FileText,
  Image as ImageIcon,
  Info,
  MapPin,
  NotebookPen,
  Paperclip,
  Send,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { BackLink } from '@/components/ui/BackLink'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Modal } from '@/components/ui/Modal'
import { TabPanel, Tabs } from '@/components/ui/Tabs'
import { useToast } from '@/components/ui/Toast'
import { ErrorState, OfflineBanner, Skeleton } from '@/components/ui/States'
import { CaseHeader } from '@/components/case/CaseHeader'
import { CaseFacts } from '@/components/case/CaseFacts'
import { ProgressTimeline } from '@/components/case/ProgressTimeline'
import { WeeklyUpdates } from '@/components/case/WeeklyUpdates'
import { EvidenceGallery } from '@/components/case/EvidenceGallery'
import { EvidenceUpload } from '@/components/case/EvidenceUpload'
import { ReviewHistory, ReviewNextStep, ReviewNotice } from '@/components/case/CaseReviewTrail'
import { CaseTracker } from '@/components/cases/CaseTracker'
import { useCaseActions, useCaseDetail } from '@/hooks/useCases'
import { useCaseReview } from '@/hooks/useCaseReview'
import { useAddProgress, useCaseProgress } from '@/hooks/useProgress'
import { useCaseEvidence, useUploadEvidence, useVerifyEvidence } from '@/hooks/useEvidence'
import { useAuth } from '@/auth/AuthContext'
import { ApiError } from '@/lib/apiClient'
import { canAddProgress, canSubmitForReview, isAwaitingDispatch, isInReviewPhase, statusMeta } from '@/lib/status'

const PROGRESS_ACTIONS = [
  { value: 'progress', label: 'Progress update' },
  { value: 'checkpoint', label: 'Checkpoint reached' },
  { value: 'interview', label: 'Witness interview' },
  { value: 'patrol', label: 'Patrol / sweep' },
  { value: 'note', label: 'Note' },
]

type TabId = 'details' | 'progress' | 'weekly' | 'evidence'

/**
 * Officer case workspace.
 *
 * One case, four lenses: what it is (details), what has been done (progress),
 * how the work breaks down by week, and what proves it (evidence). Lifecycle
 * actions are gated to the transitions the backend actually permits, and each
 * disabled action says why.
 */
export function OfficerCasePage() {
  const { id } = useParams<{ id: string }>()
  const { notify } = useToast()
  const { user } = useAuth()

  const detail = useCaseDetail(id)
  const progressQuery = useCaseProgress(id)
  const evidenceQuery = useCaseEvidence(id)
  const { dispatch, arrive, submitForReview } = useCaseActions(id)
  const review = useCaseReview(id)
  const addProgress = useAddProgress(id)
  const uploadEvidence = useUploadEvidence(id)
  const verifyEvidence = useVerifyEvidence(id)

  const [tab, setTab] = useState<TabId>('details')
  const [submitOpen, setSubmitOpen] = useState(false)
  const [finalReport, setFinalReport] = useState('')
  const [submitConflict, setSubmitConflict] = useState<string | null>(null)
  const [evidenceOpen, setEvidenceOpen] = useState(false)
  const [action, setAction] = useState(PROGRESS_ACTIONS[0].value)
  const [description, setDescription] = useState('')

  const caseItem = detail.data?.case
  const progress = progressQuery.data ?? []
  const evidence = evidenceQuery.data ?? []

  const ownsCase = Boolean(caseItem?.assignedTo && user?.id && caseItem.assignedTo === user.id)

  const networkIssue =
    ApiError.isNetwork(detail.error) ||
    ApiError.isNetwork(progressQuery.error) ||
    ApiError.isNetwork(evidenceQuery.error)

  const offline = networkIssue && Boolean(caseItem)

  const assignedOfficerName = useMemo(() => {
    const entry = detail.data?.timeline.find((t) => t.user && t.user.id === caseItem?.assignedTo)
    if (entry?.user) return `${entry.user.firstName} ${entry.user.lastName}`.trim()
    return undefined
  }, [detail.data, caseItem])

  const tabs = useMemo(
    () => [
      { id: 'details', label: 'Details', icon: <FileText className="size-4" aria-hidden /> },
      {
        id: 'progress',
        label: 'Progress',
        count: progress.length,
        icon: <NotebookPen className="size-4" aria-hidden />,
      },
      { id: 'weekly', label: 'Weekly', icon: <CalendarRange className="size-4" aria-hidden /> },
      {
        id: 'evidence',
        label: 'Evidence',
        count: evidence.length,
        icon: <ImageIcon className="size-4" aria-hidden />,
      },
    ],
    [progress.length, evidence.length],
  )

  function reportError(cause: unknown, fallback: string) {
    if (ApiError.isNetwork(cause)) {
      notify('No connection — the change was not saved.', 'error')
      return
    }
    notify(cause instanceof ApiError ? cause.message : fallback, 'error')
  }

  if (detail.isLoading) {
    return (
      <div className="mx-auto w-full max-w-4xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-64 w-full rounded-panel" />
      </div>
    )
  }

  if (detail.isError || !caseItem) {
    return (
      <div className="mx-auto w-full max-w-4xl p-4 sm:p-6">
        <BackLink to="/officer/queue" label="Back to queue" />
        <Card className="mt-4">
          <ErrorState
            title={ApiError.isNetwork(detail.error) ? 'You are offline' : 'Case unavailable'}
            description={
              ApiError.isNetwork(detail.error)
                ? 'Reconnect to load this case.'
                : 'This case does not exist, or your unit does not have access to it.'
            }
            offline={ApiError.isNetwork(detail.error)}
            onRetry={() => void detail.refetch()}
          />
        </Card>
      </div>
    )
  }

  const status = caseItem.status
  const closed = status === 'closed'
  const progressAllowed = canAddProgress(status)

  return (
    <div className="mx-auto w-full max-w-4xl p-4 sm:p-6">
      <BackLink to="/officer/queue" label="Back to queue" />

      {offline ? (
        <div className="mb-3">
          <OfflineBanner />
        </div>
      ) : null}

      <div className="mt-3 flex flex-col gap-4">
        <CaseHeader
          caseItem={caseItem}
          assignedOfficerName={assignedOfficerName}
          actions={
            <>
              {status === 'assigned' ? (
                <Button
                  variant="primary"
                  loading={dispatch.isPending}
                  icon={<BellRing className="size-4" aria-hidden />}
                  onClick={() =>
                    dispatch.mutate(undefined, {
                      onSuccess: () => notify('Marked as dispatched.', 'success'),
                      onError: (cause) => reportError(cause, 'Could not dispatch this case.'),
                    })
                  }
                >
                  Dispatch
                </Button>
              ) : null}

              {status === 'dispatched' ? (
                <Button
                  variant="primary"
                  loading={arrive.isPending}
                  icon={<MapPin className="size-4" aria-hidden />}
                  onClick={() =>
                    arrive.mutate(undefined, {
                      onSuccess: () => notify('Marked as on scene.', 'success'),
                      onError: (cause) => reportError(cause, 'Could not update this case.'),
                    })
                  }
                >
                  Mark on scene
                </Button>
              ) : null}

              {canSubmitForReview(status) ? (
                <Button
                  variant="primary"
                  icon={<Send className="size-4" aria-hidden />}
                  onClick={() => {
                    setSubmitConflict(null)
                    setFinalReport(caseItem.finalReport ?? '')
                    setSubmitOpen(true)
                  }}
                >
                  Submit for review
                </Button>
              ) : null}

              {status === 'pending_admin_review' ? (
                <Badge tone="neutral" className="h-9 px-3">
                  With an administrator for a closure decision
                </Badge>
              ) : null}

              {status === 'pending' ? (
                <Badge tone="neutral" className="h-9 px-3">
                  Awaiting assignment by your unit admin
                </Badge>
              ) : null}
            </>
          }
        />

        {/* Progress tracker — the officer and the reporter look at the same rail. */}
        <Card className="space-y-2 p-5">
          <h2 className="text-sm font-semibold text-ink">Case progress</h2>
          <CaseTracker caseItem={caseItem} timeline={detail.data?.timeline} />
        </Card>

        <ReviewNotice status={status} reviews={review.data?.reviews ?? []} />

        {submitConflict ? (
          <div className="flex items-start gap-2.5 rounded-panel border border-warn/30 bg-warn/5 p-3">
            <Info className="mt-0.5 size-4 shrink-0 text-warn" aria-hidden />
            <p className="min-w-0 flex-1 text-sm text-ink">{submitConflict}</p>
            <Button size="sm" variant="ghost" onClick={() => setSubmitConflict(null)}>
              Dismiss
            </Button>
          </div>
        ) : null}

        <Card>
          <Tabs items={tabs} activeId={tab} onChange={(next) => setTab(next as TabId)} />

          {tab === 'details' ? (
            <TabPanel id="details">
              <CaseFacts
                caseItem={caseItem}
                timeline={detail.data?.timeline ?? []}
                feedback={detail.data?.feedback ?? []}
              />
              <section className="border-t border-border p-4">
                <h3 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                  Closure review
                </h3>
                <ReviewNextStep status={status} />
                {review.isError ? (
                  <p className="mt-2 text-xs text-ink-muted">
                    The review history did not load. It may be a connection problem.
                  </p>
                ) : (
                  <ReviewHistory
                    reviews={review.data?.reviews ?? []}
                    status={status}
                    isLoading={review.isLoading}
                    className="mt-3"
                  />
                )}
              </section>
            </TabPanel>
          ) : null}

          {tab === 'progress' ? (
            <TabPanel id="progress">
              <div className="border-b border-border p-4">
                {progressAllowed ? (
                  <form
                    className="flex flex-col gap-3"
                    onSubmit={(event) => {
                      event.preventDefault()
                      const text = description.trim()
                      if (!text) return
                      addProgress.mutate(
                        { action, description: text },
                        {
                          onSuccess: () => {
                            setDescription('')
                            notify('Progress update added.', 'success')
                          },
                          onError: (cause) => reportError(cause, 'Could not add the update.'),
                        },
                      )
                    }}
                  >
                    <div className="grid gap-3 sm:grid-cols-[minmax(0,12rem)_1fr]">
                      <Field label="Kind of update">
                        {(props) => (
                          <Select
                            {...props}
                            value={action}
                            onChange={(event) => setAction(event.target.value)}
                          >
                            {PROGRESS_ACTIONS.map((option) => (
                              <option key={option.value} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </Select>
                        )}
                      </Field>
                      <Field label="What happened" required>
                        {(props) => (
                          <Input
                            {...props}
                            value={description}
                            onChange={(event) => setDescription(event.target.value)}
                            placeholder="Brief, specific, in plain language"
                          />
                        )}
                      </Field>
                    </div>
                    <div className="flex justify-end">
                      <Button
                        type="submit"
                        variant="primary"
                        size="sm"
                        loading={addProgress.isPending}
                        disabled={!description.trim()}
                      >
                        Add update
                      </Button>
                    </div>
                  </form>
                ) : (
                  <p className="flex items-start gap-2 text-xs text-ink-muted">
                    <Info className="mt-0.5 size-4 shrink-0 text-signal" aria-hidden />
                    {progressGateMessage(status)}
                  </p>
                )}
              </div>

              {progressQuery.isLoading ? (
                <ProgressTimeline progress={[]} isLoading />
              ) : progressQuery.isError ? (
                <ErrorState
                  title="Could not load progress"
                  description="The updates for this case did not load."
                  onRetry={() => void progressQuery.refetch()}
                />
              ) : (
                <ProgressTimeline progress={progress} />
              )}
            </TabPanel>
          ) : null}

          {tab === 'weekly' ? (
            <TabPanel id="weekly">
              <WeeklyUpdates
                caseId={id}
                canFile={ownsCase && !closed}
                currentUserId={user?.id}
                progress={progress}
                timeline={detail.data?.timeline ?? []}
                showCitizenVisibility
              />
            </TabPanel>
          ) : null}

          {tab === 'evidence' ? (
            <TabPanel id="evidence">
              <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
                <p className="text-xs text-ink-muted">
                  {closed
                    ? 'This case is closed — no further evidence can be attached.'
                    : 'Attach photos, recordings and documents, then verify them.'}
                </p>
                <Button
                  size="sm"
                  variant="secondary"
                  icon={<Paperclip className="size-4" aria-hidden />}
                  disabled={closed}
                  title={closed ? 'Evidence cannot be added to a closed case.' : undefined}
                  onClick={() => setEvidenceOpen(true)}
                >
                  Attach
                </Button>
              </div>

              {evidenceQuery.isLoading ? (
                <EvidenceGallery evidence={[]} isLoading />
              ) : evidenceQuery.isError ? (
                <ErrorState
                  title="Could not load evidence"
                  description="The evidence for this case did not load."
                  onRetry={() => void evidenceQuery.refetch()}
                />
              ) : (
                <EvidenceGallery
                  evidence={evidence}
                  canVerify={!closed}
                  verifyingId={verifyEvidence.isPending ? verifyEvidence.variables : null}
                  onVerify={(evidenceId) =>
                    verifyEvidence.mutate(evidenceId, {
                      onSuccess: () => notify('Evidence verified.', 'success'),
                      onError: (cause) => reportError(cause, 'Could not verify that item.'),
                    })
                  }
                />
              )}
            </TabPanel>
          ) : null}
        </Card>
      </div>

      <Modal
        open={submitOpen}
        onClose={() => setSubmitOpen(false)}
        title="Submit this case for closure review"
        description="The final report is what the administrator judges. They can approve the closure, or send the case back to you with a comment."
        footer={
          <>
            <Button variant="ghost" onClick={() => setSubmitOpen(false)}>
              Keep editing
            </Button>
            <Button
              variant="primary"
              loading={submitForReview.isPending}
              disabled={!finalReport.trim()}
              onClick={() =>
                submitForReview.mutate(finalReport.trim(), {
                  onSuccess: () => {
                    setSubmitOpen(false)
                    setFinalReport('')
                    notify('Submitted for review.', 'success')
                  },
                  onError: (cause) => {
                    setSubmitOpen(false)
                    const moved = conflictStatus(cause)
                    if (moved) {
                      setSubmitConflict(describeMovedStatus(moved))
                      return
                    }
                    reportError(cause, 'Could not submit this case for review.')
                  },
                })
              }
            >
              Submit for review
            </Button>
          </>
        }
      >
        <Field
          label="Final report"
          required
          hint="What was found, what was done, and how the case was resolved. This becomes part of the permanent record."
        >
          {(props) => (
            <Textarea
              {...props}
              rows={8}
              value={finalReport}
              onChange={(event) => setFinalReport(event.target.value)}
              placeholder="Summary of the investigation and its outcome…"
            />
          )}
        </Field>

        {!finalReport.trim() ? (
          <p className="mt-2 flex items-start gap-2 text-xs text-ink-muted">
            <Info className="mt-0.5 size-4 shrink-0 text-signal" aria-hidden />
            Add the final report first — an administrator cannot approve a closure without one.
          </p>
        ) : null}
      </Modal>

      <EvidenceUpload
        open={evidenceOpen}
        onClose={() => setEvidenceOpen(false)}
        submitting={uploadEvidence.isPending}
        caseLatitude={caseItem.latitude}
        caseLongitude={caseItem.longitude}
        caseLocationLabel={caseItem.location}
        onSubmit={(input) =>
          uploadEvidence.mutate(input, {
            onSuccess: () => {
              setEvidenceOpen(false)
              notify('Evidence attached.', 'success')
            },
            onError: (cause) => reportError(cause, 'Could not attach that evidence.'),
          })
        }
      />
    </div>
  )
}

function progressGateMessage(status: string): string {
  if (status === 'closed') return 'This case is closed — the progress record is now read-only.'
  if (isAwaitingDispatch(status)) {
    return 'Progress updates can be added once the case is dispatched. This case has not been dispatched yet.'
  }
  if (status === 'admin_changes_requested') {
    return 'An administrator asked for changes to this case, so the progress record is read-only. Revise the final report and submit it for review again.'
  }
  if (isInReviewPhase(status)) {
    return 'This case is with an administrator for a closure decision, so the progress record is read-only until they decide.'
  }
  const meta = statusMeta(status)
  return meta.known
    ? `Progress updates cannot be added while a case is ${meta.label.toLowerCase()}.`
    : `Progress updates cannot be added while a case is in a state this app does not recognise (${status}).`
}

function conflictStatus(cause: unknown): string | undefined {
  if (!(cause instanceof ApiError) || cause.status !== 409) return undefined
  const body = cause.body as { status?: unknown } | undefined
  return typeof body?.status === 'string' ? body.status : undefined
}

function describeMovedStatus(moved: string): string {
  const meta = statusMeta(moved)
  const label = meta.known
    ? meta.label.toLowerCase()
    : `in a state this app does not recognise (${moved})`
  return `This case is now ${label}, so it can no longer be submitted for review. This screen has been refreshed.`
}