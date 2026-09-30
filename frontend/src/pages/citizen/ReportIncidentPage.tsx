import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  Image,
  Link2,
  MapPin,
  Paperclip,
  Plus,
  Send,
  Trash2,
  X,
} from 'lucide-react'
import { MapView } from '@/components/map/MapView'
import { BackLink } from '@/components/ui/BackLink'
import { Button } from '@/components/ui/Button'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Badge, PriorityChip, StatusChip } from '@/components/ui/Chips'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { caseKeys, useCreateCase } from '@/hooks/useCases'
import { evidenceKeys } from '@/hooks/useEvidence'
import { useUnits, useNearbyUnits } from '@/hooks/useUnits'
import { useLocation } from '@/hooks/useLocation'
import { useReverseGeocode, formatAddress } from '@/hooks/useReverseGeocode'
import { ApiError, api } from '@/lib/apiClient'
import { cn } from '@/lib/cn'
import { formatCoord } from '@/lib/format'
import {
  clearReportDraft,
  emptyReportDraft,
  isReportDraftEmpty,
  loadReportDraft,
  reportDraftStorage,
  saveReportDraft,
  type DraftEvidenceLink,
  type ReportDraft,
} from '@/lib/reportDraft'
import type { CreateCaseInput, CreateCaseResponse, EvidenceCreateResponse, PresignEvidenceResponse } from '@/types/api'

interface PhotoFile {
  file: File
  preview: string
  type: string
}

/**
 * Filing a report — the citizen's one long form, and the only place this app
 * *creates* a case.
 *
 * Four steps, in the order the answers depend on each other: what happened, where
 * it is (which decides who could respond), optional evidence, then a review of the
 * whole thing before anything is sent.
 *
 * Three constraints from the contract shape the design, and each one is why a
 * particular piece of copy exists:
 *
 * 1. **There is no category field on `POST /cases`.** So the wizard does not ask
 *    for one, and no screen may read it back. What the unit triages on is the
 *    title and the description — which is why they are the two required fields
 *    and why the disabled-Continue reasons name them individually.
 * 2. **A case with no unit is invisible to every officer and unit admin.**
 *    `GetAllCases` scopes an officer's list by `unit_id`, so a zero unit matches
 *    nobody's query. Selecting a unit is therefore *required* here, even though
 *    the endpoint itself treats it as optional — offering an option whose
 *    consequence is "no one will be shown your report" would be a lie by omission.
 * 3. **Evidence is attached after the case exists**, because
 *    `POST /evidence/upload` needs a `caseId`. The reporter's links are collected
 *    in step 3 but sent in step 4's wake, so a link that fails is reported
 *    against a receipt the reporter already has — never as a lost report.
 *
 * Deliberately absent: an SOS path. `isSOS` bands P1 and triggers an immediate
 * dispatch attempt, and that belongs to an arming flow with its own confirm step,
 * not to a checkbox on a form someone is filling in while something is happening.
 */

const STEPS = [
  { key: 'what', label: 'What happened' },
  { key: 'where', label: 'Where' },
  { key: 'evidence', label: 'Evidence' },
  { key: 'review', label: 'Review' },
] as const

type StepKey = (typeof STEPS)[number]['key']

/** Conventional `type` values for the free-form field on `/evidence/upload`. */
const EVIDENCE_TYPES = [
  { value: 'image', label: 'Photo' },
  { value: 'video', label: 'Video' },
  { value: 'audio', label: 'Audio recording' },
  { value: 'document', label: 'Document' },
] as const

const MAX_EVIDENCE_LINKS = 3

/**
 * A link has to be a link.
 *
 * `fileUrl` is free-form on the backend, so a mistyped word would be accepted,
 * stored, and handed to an officer who cannot open it. Requiring http(s) also
 * keeps a `javascript:` URL out of the record. The value is never rendered as an
 * anchor in this flow for the same reason — it is shown as text.
 */
function linkError(value: string): string | undefined {
  const trimmed = value.trim()
  if (trimmed === '') return undefined
  if (!/^https?:\/\//i.test(trimmed)) return 'Must be a link starting with http:// or https://'
  return undefined
}

function errorMessage(error: unknown): string {
  if (ApiError.isNetwork(error)) return 'Could not reach the server. Check your connection.'
  if (error instanceof ApiError) return error.message
  return 'Something went wrong.'
}

export function ReportIncidentPage() {
  // Resolved once, and guarded: even reading the `localStorage` global can throw
  // where storage is blocked.
  const [storage] = useState(reportDraftStorage)
  const [restored] = useState(() => loadReportDraft(storage))

  const [draft, setDraft] = useState<ReportDraft>(() => restored ?? emptyReportDraft())
  const [stepIndex, setStepIndex] = useState(0)
  const [showRestoredNotice, setShowRestoredNotice] = useState(
    () => restored !== null && !isReportDraftEmpty(restored),
  )
  const [created, setCreated] = useState<CreateCaseResponse | null>(null)
  const [attachments, setAttachments] = useState<AttachmentState[]>([])
  const [submitError, setSubmitError] = useState<string | null>(null)
  // Photo files selected in the evidence step (not persisted to draft)
  const [photos, setPhotos] = useState<PhotoFile[]>([])

  const createCase = useCreateCase()
  const queryClient = useQueryClient()
  const { notify } = useToast()

  const { latitude: userLat, longitude: userLng } = useLocation()
  const geoLat = draft.latitude ?? userLat
  const geoLng = draft.longitude ?? userLng
  const { data: geo, isLoading: geoLoading } = useReverseGeocode(geoLat, geoLng)

  const step = STEPS[stepIndex]

  /**
   * Generate SHA-256 hash of a file for the presign request.
   */
  async function sha256(file: File): Promise<string> {
    const buffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
    return Array.from(new Uint8Array(hashBuffer))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('')
  }

  /**
   * Presign → PUT → Confirm flow for a single photo file.
   */
  async function uploadPhoto(caseId: string, photo: PhotoFile): Promise<string> {
    const hash = await sha256(photo.file)
    const presign = await api.post<PresignEvidenceResponse>(`/evidence/case/${caseId}/presign`, {
      filename: photo.file.name,
      contentType: photo.file.type,
      sizeBytes: photo.file.size,
      sha256: hash,
    })
    // PUT the file to the presigned URL (no auth header — presigned URL carries auth)
    const putResponse = await fetch(presign.uploadUrl, {
      method: 'PUT',
      // Go signs these S3 metadata headers in PresignPut. The PUT must send
      // the same values or the storage service rejects its signature.
      headers: {
        'Content-Type': photo.file.type,
        'x-amz-meta-sha256': hash,
        'x-amz-meta-original-filename': photo.file.name,
        'x-amz-meta-case-id': caseId,
      },
      body: photo.file,
    })
    if (!putResponse.ok) {
      throw new Error(`Upload failed: ${putResponse.statusText}`)
    }
    // Confirm the upload
    await api.post(`/evidence/case/${caseId}/confirm`, {
      key: presign.key,
      filename: photo.file.name,
      contentType: photo.file.type,
      sizeBytes: photo.file.size,
    })
    return presign.key
  }

  /**
   * Attach photos after the case is created. Best-effort — failures are reported
   * via toast but do not fail the report.
   */
  async function attachPhotos(caseId: string, photoFiles: PhotoFile[]) {
    const linkCount = linksToSend.length
    for (const [index, photo] of photoFiles.entries()) {
      const attachmentIndex = linkCount + index
      setAttachments((current) =>
        current.map((item, i) => (i === attachmentIndex ? { ...item, status: 'sending' } : item)),
      )
      try {
        await uploadPhoto(caseId, photo)
        setAttachments((current) =>
          current.map((item, i) => (i === attachmentIndex ? { ...item, status: 'done' } : item)),
        )
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Upload failed'
        setAttachments((current) =>
          current.map((item, i) =>
            i === attachmentIndex ? { ...item, status: 'failed', message } : item,
          ),
        )
        notify(`Photo failed to upload: ${photo.file.name}`, 'error')
      }
    }
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: evidenceKeys.forCase(caseId) }),
      queryClient.invalidateQueries({ queryKey: caseKeys.detail(caseId) }),
      queryClient.invalidateQueries({ queryKey: caseKeys.list() }),
    ])
  }

  /**
   * Written on every change, synchronously.
   *
   * No debounce: a draft is a few hundred bytes, and a timer would need a flush-on-
   * unmount path to avoid dropping the last keystrokes — more code than the write
   * it saves. This is also why an emptied form removes the key outright.
   */
  useEffect(() => {
    saveReportDraft(storage, draft)
  }, [storage, draft])

  // Move focus to the step heading on a step change, so a keyboard or screen
  // reader lands inside the new step instead of at the top of the document.
  // Skipped on mount: nothing has changed yet, and stealing focus then scrolls.
  const headingRef = useRef<HTMLHeadingElement>(null)
  const mounted = useRef(false)
  useEffect(() => {
    if (!mounted.current) {
      mounted.current = true
      return
    }
    headingRef.current?.focus()
  }, [stepIndex])

  // Prefill location from reverse geocode when the field is empty and we have a result
  useEffect(() => {
    if (geo && draft.location.trim() === '' && !geoLoading) {
      update('location', formatAddress(geo))
    }
  }, [geo, geoLoading, draft.location, update])

  function update<K extends keyof ReportDraft>(key: K, value: ReportDraft[K]) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  function placePin(latitude: number, longitude: number) {
    setDraft((current) => ({ ...current, latitude, longitude }))
  }

  const linkErrors = useMemo(
    () => draft.evidence.map((link) => linkError(link.fileUrl)),
    [draft.evidence],
  )
  const linksToSend = draft.evidence.filter((link) => link.fileUrl.trim() !== '')

  /** The one thing standing between this step and the next, or `null`. */
  function blockedReason(key: StepKey): string | null {
    if (key === 'what') {
      if (draft.title.trim() === '') return 'Add a title — it is the line the unit reads first.'
      if (draft.description.trim() === '')
        return 'Describe what is happening. This is what the unit triages on.'
      return null
    }
    if (key === 'where') {
      if (draft.latitude === null || draft.longitude === null)
        return 'Place the pin on the map so the unit knows where to go.'
      if (!draft.location.trim())
        return 'Add a landmark or address. The Go reporting endpoint requires a location.'
      if (draft.unitId === null)
        return 'Choose a unit. A report sent without one is not shown in any unit’s queue.'
      return null
    }
    if (key === 'evidence') {
      if (linkErrors.some(Boolean)) return 'Fix the link marked below, or remove it.'
      return null
    }
    return null
  }

  const stepBlocked = blockedReason(step.key)
  const submitBlocked =
    blockedReason('what') ?? blockedReason('where') ?? blockedReason('evidence')

  /**
   * Create the case, then attach the links and photos, in that order.
   *
   * The draft is cleared the moment the create succeeds — *before* any evidence
   * is attempted — so a failed attachment can never leave a reporter able to
   * submit the same report twice.
   */
  async function submit() {
    setSubmitError(null)
    const invalid = blockedReason('what') ?? blockedReason('where') ?? blockedReason('evidence')
    if (invalid) {
      setSubmitError(invalid)
      return
    }

    const input: CreateCaseInput = {
      title: draft.title.trim(),
      description: draft.description.trim(),
      location: draft.location.trim() || undefined,
      latitude: draft.latitude ?? undefined,
      longitude: draft.longitude ?? undefined,
      unitId: draft.unitId ?? undefined,
      // Only the literal "high" is honoured; anything else is P3. Never `isSOS`.
      priority: draft.urgent ? 'high' : undefined,
      // Sent only when checked; omitted otherwise, so the backend default (false)
      // applies and a normal (signed) report is created.
      hideLocation: draft.hideLocation || undefined,
    }

    let response: CreateCaseResponse
    try {
      response = await createCase.mutateAsync(input)
    } catch (error) {
      // Nothing was created. Say so, and keep every answer on screen.
      setSubmitError(errorMessage(error))
      return
    }

    clearReportDraft(storage)
    setCreated(response)

    // Build attachment state for both links and photos
    const linkAttachments = linksToSend.map((link) => ({ ...link, status: 'queued' as const }))
    const photoAttachments = photos.map((photo) => ({
      fileUrl: photo.file.name,
      type: photo.type,
      status: 'queued' as const,
      isPhoto: true,
      preview: photo.preview,
    }))
    setAttachments([...linkAttachments, ...photoAttachments])

    // Attach links (existing flow)
    void attachEvidence(response.case.id, linksToSend)
    // Attach photos (new flow) — best effort, non-blocking
    void attachPhotos(response.case.id, photos)
  }

  /**
   * Attach the reporter's links one at a time.
   *
   * This calls the endpoint `useUploadEvidence` wraps rather than that hook: the
   * hook is one mutation with one terminal state, and this needs *per-link*
   * outcomes on a single receipt, against a case id that only exists mid-flight.
   *
   * Sequential on purpose. Three parallel uploads on a weak connection make all
   * three slow and none of them legible; one at a time gives an honest progress
   * count and stops at the first failure if the connection is gone.
   */
  async function attachEvidence(caseId: string, links: DraftEvidenceLink[]) {
    for (const [index, link] of links.entries()) {
      setAttachments((current) =>
        current.map((item, i) => (i === index ? { ...item, status: 'sending' } : item)),
      )
      try {
        await api.post<EvidenceCreateResponse>('/evidence/upload', {
          caseId,
          type: link.type || 'image',
          fileUrl: link.fileUrl.trim(),
        })
        setAttachments((current) =>
          current.map((item, i) => (i === index ? { ...item, status: 'done' } : item)),
        )
      } catch (error) {
        setAttachments((current) =>
          current.map((item, i) =>
            i === index ? { ...item, status: 'failed', message: errorMessage(error) } : item,
          ),
        )
      }
    }

    // Keep the officer's view of this case honest: the evidence list and the case
    // detail both carry one, and both were fetched before these links existed.
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: evidenceKeys.forCase(caseId) }),
      queryClient.invalidateQueries({ queryKey: caseKeys.detail(caseId) }),
      queryClient.invalidateQueries({ queryKey: caseKeys.list() }),
    ])
  }

  async function retryAttachment(caseId: string, index: number) {
    const link = attachments[index]
    if (!link) return
    setAttachments((current) =>
      current.map((item, i) => (i === index ? { ...item, status: 'sending', message: undefined } : item)),
    )
    try {
      if (link.isPhoto) {
        const photo = photos[index - linksToSend.length]
        if (!photo) throw new Error('Photo is no longer available. Please file a new report to attach it.')
        await uploadPhoto(caseId, photo)
      } else {
        await api.post<EvidenceCreateResponse>('/evidence/upload', {
          caseId,
          type: link.type || 'image',
          fileUrl: link.fileUrl.trim(),
        })
      }
      setAttachments((current) =>
        current.map((item, i) => (i === index ? { ...item, status: 'done' } : item)),
      )
      await queryClient.invalidateQueries({ queryKey: evidenceKeys.forCase(caseId) })
    } catch (error) {
      setAttachments((current) =>
        current.map((item, i) =>
          i === index ? { ...item, status: 'failed', message: errorMessage(error) } : item,
        ),
      )
    }
  }

  function startAnother() {
    clearReportDraft(storage)
    setDraft(emptyReportDraft())
    setCreated(null)
    setAttachments([])
    setSubmitError(null)
    setStepIndex(0)
  }

  async function copyTrackingId(id: string) {
    try {
      await navigator.clipboard.writeText(id)
      notify('Tracking ID copied', 'success')
    } catch {
      notify('Could not copy — select the ID and copy it by hand.', 'info')
    }
  }

  if (created) {
    return (
      <Receipt
        created={created}
        attachments={attachments}
        onCopy={copyTrackingId}
        onRetry={(index) => void retryAttachment(created.case.id, index)}
        onStartAnother={startAnother}
      />
    )
  }

  return (
    <div className="mx-auto w-full max-w-2xl p-4 sm:p-6">
      <BackLink to="/" label="Back to your reports" />

      <header className="mt-3 mb-4">
        <h1 className="text-xl font-semibold text-ink">Report an incident</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Four short steps. Nothing is sent until you confirm on the last one.
        </p>
      </header>

      {showRestoredNotice ? (
        <div className="mb-4 flex items-start gap-2 rounded-lg border border-signal/30 bg-signal/10 px-3 py-2 text-xs text-signal">
          <Paperclip className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          <p className="flex-1">
            We kept the answers you had started. Nothing has been sent.
          </p>
          <button
            type="button"
            onClick={() => setShowRestoredNotice(false)}
            className="shrink-0 rounded p-0.5 hover:bg-signal/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
            aria-label="Dismiss"
          >
            <X className="size-3.5" aria-hidden />
          </button>
        </div>
      ) : null}

      <div className="mb-3 rounded-panel glass-panel px-3 py-2">
        <ol className="flex items-center gap-1.5" role="list">
          {STEPS.map((item, index) => {
            const state = index < stepIndex ? 'done' : index === stepIndex ? 'current' : 'upcoming'
            const canJump = index < stepIndex
            return (
              <li key={item.key} className="flex flex-1 items-center gap-1.5">
                <button
                  type="button"
                  disabled={!canJump}
                  onClick={() => { if (canJump) setStepIndex(index) }}
                  aria-current={state === 'current' ? 'step' : undefined}
                  aria-label={`Step ${index + 1}: ${item.label}`}
                  className={cn(
                    'flex items-center gap-1.5 rounded-md px-1.5 py-1 transition-colors',
                    canJump && 'cursor-pointer hover:bg-signal/10',
                    !canJump && 'cursor-default',
                  )}
                >
                  <span
                    aria-hidden
                    className={cn(
                      'grid size-6 shrink-0 place-items-center rounded-full text-[11px] font-semibold tabular-nums',
                      state === 'done' && 'bg-signal text-signal-ink',
                      state === 'current' && 'border-2 border-signal bg-signal/15 text-signal',
                      state === 'upcoming' && 'border border-border-hi text-ink-faint',
                    )}
                  >
                    {state === 'done' ? <CheckCircle2 className="size-3.5" /> : index + 1}
                  </span>
                  <span
                    className={cn(
                      'hidden text-[11px] font-medium sm:inline',
                      state === 'current' && 'text-ink',
                      state === 'done' && 'text-ink-muted',
                      state === 'upcoming' && 'text-ink-faint',
                    )}
                  >
                    {item.label}
                  </span>
                </button>
                {index < STEPS.length - 1 ? (
                  <span
                    aria-hidden
                    className={cn('h-px flex-1', index < stepIndex ? 'bg-signal/60' : 'bg-border-hi')}
                  />
                ) : null}
              </li>
            )
          })}
        </ol>
      </div>

      <Card>
        <CardHeader title={step.label} />
        <CardBody className="flex flex-col gap-5">
          <h2 ref={headingRef} tabIndex={-1} className="sr-only focus:outline-none">
            Step {stepIndex + 1}: {step.label}
          </h2>

          {step.key === 'what' ? (
            <div className="flex flex-col gap-4">
              <Field
                label="What is happening?"
                required
                hint="Describe what you can see. A description is enough — avoid naming people."
              >
                {({ id, ...aria }) => (
                  <Input
                    id={id}
                    {...aria}
                    value={draft.title}
                    maxLength={120}
                    onChange={(event) => update('title', event.target.value)}
                    placeholder="Burst pipe flooding the junction"
                  />
                )}
              </Field>

              <Field
                label="Tell us more"
                required
                hint="What you saw, when it started, and anything the unit should know before arriving."
              >
                {({ id, ...aria }) => (
                  <Textarea
                    id={id}
                    {...aria}
                    rows={5}
                    value={draft.description}
                    maxLength={1000}
                    onChange={(event) => update('description', event.target.value)}
                    placeholder="Water has been running since yesterday evening and is now over the kerb."
                  />
                )}
              </Field>

              {/* Amber, not red: urgent triage is not the emergency path, and red
                  is reserved for SOS alone. */}
              <label
                htmlFor="report-urgent"
                className={cn(
                  'flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors',
                  draft.urgent ? 'border-warn/40 bg-warn/10' : 'border-border-hi bg-surface-hi',
                )}
              >
                <input
                  id="report-urgent"
                  type="checkbox"
                  checked={draft.urgent}
                  onChange={(event) => update('urgent', event.target.checked)}
                  className="mt-0.5 size-4 shrink-0 accent-warn"
                />
                <span>
                  <span className="block text-sm font-medium text-ink">
                    Someone is in danger or injured now
                  </span>
                  <span className="mt-0.5 block text-xs text-ink-muted">
                    This marks the report urgent so it is triaged first. It does not send anyone —
                    if this is happening right now, call your local emergency number.
                  </span>
                </span>
              </label>
            </div>
          ) : null}

          {step.key === 'where' ? (
          <WhereStep
            draft={draft}
            onLocationChange={(value) => update('location', value)}
            onPick={placePin}
            onClearPin={() =>
              setDraft((current) => ({ ...current, latitude: null, longitude: null }))
            }
            onHideLocationChange={(value) => update('hideLocation', value)}
            onUnitChange={(unitId) => update('unitId', unitId)}
            geoLoading={geoLoading}
            userLat={userLat}
            userLng={userLng}
            onUseCurrentLocation={() => {
              if (userLat != null && userLng != null) {
                placePin(userLat, userLng)
                if (geo) update('location', formatAddress(geo))
              }
            }}
          />
          ) : null}

          {step.key === 'evidence' ? (
            <div className="flex flex-col gap-4">
              <p className="text-xs text-ink-muted">
                Photographs or documents help the unit understand the report before they arrive.
                Links and photos are optional, and you can send the report without them.
              </p>

              {/* Photo upload section */}
              <div className="rounded-lg border border-dashed border-border-hi bg-surface-hi p-3">
                <label className="cursor-pointer">
                  <input
                    type="file"
                    accept="image/*"
                    multiple
                    onChange={(event) => {
                      const files = Array.from(event.target.files ?? [])
                      const validFiles = files
                        .filter((file) => file.type.startsWith('image/') && file.size <= 5 * 1024 * 1024)
                        .slice(0, 3 - photos.length)
                      if (validFiles.length !== files.length) {
                        notify('Some files were skipped: only images up to 5 MB, max 3 total', 'error')
                      }
                      validFiles.forEach((file) => {
                        const preview = URL.createObjectURL(file)
                        setPhotos((current) => [...current, { file, preview, type: 'image' }])
                      })
                      event.currentTarget.value = ''
                    }}
                    className="sr-only"
                    id="photo-upload"
                    disabled={photos.length >= 3}
                  />
                  <div className="flex flex-wrap gap-2">
                    {photos.map((photo, index) => (
                      <div key={index} className="relative size-20 rounded-lg overflow-hidden">
                        <img src={photo.preview} alt="Preview" className="size-full object-cover" />
                        <button
                          type="button"
                          onClick={() => {
                            URL.revokeObjectURL(photo.preview)
                            setPhotos((current) => current.filter((_, i) => i !== index))
                          }}
                          className="absolute top-1 right-1 size-5 rounded-full bg-black/60 text-white flex items-center justify-center hover:bg-black/80"
                          aria-label="Remove photo"
                        >
                          <X className="size-3" aria-hidden />
                        </button>
                      </div>
                    ))}
                    {photos.length < 3 && (
                      <div className="size-20 rounded-lg border-2 border-dashed border-border-hi flex flex-col items-center justify-center text-ink-muted hover:border-signal hover:bg-signal/10 transition-colors">
                        <Image className="size-6 mb-1" aria-hidden />
                        <span className="text-xs">Add photos</span>
                        <span className="text-[10px]">Max 3 · 5 MB</span>
                      </div>
                    )}
                  </div>
                </label>
                {photos.length > 0 && (
                  <p className="mt-2 text-[11px] text-ink-muted">
                    {photos.length}/3 photos selected
                  </p>
                )}
              </div>

              {/* Links section (existing) */}
              <div className="flex flex-col gap-4">
                <h3 className="text-xs font-medium text-ink-muted uppercase tracking-wide">Links</h3>
                {draft.evidence.length === 0 ? (
                  <p className="rounded-lg border border-dashed border-border-hi px-3 py-6 text-center text-xs text-ink-faint">
                    No links added.
                  </p>
                ) : null}

                {draft.evidence.map((link, index) => (
                  <div key={index} className="rounded-lg border border-border-hi bg-surface-hi p-3">
                    <div className="flex items-start gap-2">
                      <Field
                        className="flex-1"
                        label={`Link ${index + 1}`}
                        error={linkErrors[index]}
                      >
                        {({ id, ...aria }) => (
                          <Input
                            id={id}
                            {...aria}
                            value={link.fileUrl}
                            inputMode="url"
                            placeholder="https://…"
                            onChange={(event) =>
                              update(
                                'evidence',
                                draft.evidence.map((item, i) =>
                                  i === index ? { ...item, fileUrl: event.target.value } : item,
                                ),
                              )
                            }
                          />
                        )}
                      </Field>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="mt-5"
                        aria-label={`Remove link ${index + 1}`}
                        onClick={() =>
                          update(
                            'evidence',
                            draft.evidence.filter((_, i) => i !== index),
                          )
                        }
                      >
                        <Trash2 className="size-4" aria-hidden />
                      </Button>
                    </div>

                    <Field className="mt-3" label="What is it?">
                      {({ id }) => (
                        <Select
                          id={id}
                          value={link.type}
                          onChange={(event) =>
                            update(
                              'evidence',
                              draft.evidence.map((item, i) =>
                                i === index ? { ...item, type: event.target.value } : item,
                              ),
                            )
                          }
                        >
                          {EVIDENCE_TYPES.map((option) => (
                            <option key={option.value} value={option.value}>
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      )}
                    </Field>
                  </div>
                ))}

                {draft.evidence.length < MAX_EVIDENCE_LINKS ? (
                  <Button
                    variant="secondary"
                    size="sm"
                    icon={<Plus className="size-4" aria-hidden />}
                    onClick={() =>
                      update('evidence', [
                        ...draft.evidence,
                        { fileUrl: '', type: EVIDENCE_TYPES[0].value },
                      ])
                    }
                  >
                    Add a link
                  </Button>
                ) : null}

                <p className="flex items-start gap-1.5 text-[11px] text-ink-faint">
                  <Link2 className="mt-0.5 size-3.5 shrink-0" aria-hidden />
                  <span>
                    Up to {MAX_EVIDENCE_LINKS} links. The file has to be hosted somewhere you can
                    share a link to.
                  </span>
                </p>
              </div>
            </div>
          ) : null}

          {step.key === 'review' ? (
            <ReviewStep draft={draft} photos={photos} onEdit={setStepIndex} />
          ) : null}

          {stepBlocked && step.key !== 'review' ? (
            <p className="rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn">
              {stepBlocked}
            </p>
          ) : null}

          {step.key === 'review' && submitError ? (
            <div
              role="alert"
              className="flex items-start gap-2 rounded-lg border border-emergency/30 bg-emergency/10 px-3 py-2 text-xs text-emergency"
            >
              <AlertTriangle className="mt-0.5 size-3.5 shrink-0" aria-hidden />
              <p>
                <span className="font-semibold">Your report was not sent.</span> Nothing was
                created, and every answer is still here. {submitError}
              </p>
            </div>
          ) : null}

          <div className="sticky bottom-0 z-10 -mx-4 -mb-4 mt-2 flex flex-col-reverse gap-2 rounded-b-panel border-t border-border-hi bg-surface/90 px-4 py-3 shadow-[0_-8px_24px_rgba(0,0,0,0.20)] backdrop-blur-md sm:-mx-5 sm:flex-row sm:items-center sm:justify-between sm:px-5">
            <Button
              variant="ghost"
              onClick={() => setStepIndex((index) => Math.max(0, index - 1))}
              disabled={stepIndex === 0}
            >
              Back
            </Button>

            {step.key === 'review' ? (
              <Button
                variant="primary"
                icon={<Send className="size-4" aria-hidden />}
                loading={createCase.isPending}
                disabled={Boolean(submitBlocked)}
                onClick={() => void submit()}
                className="sm:min-w-44"
              >
                {createCase.isPending ? 'Sending…' : 'Send report'}
              </Button>
            ) : (
              <Button
                variant="primary"
                disabled={Boolean(stepBlocked)}
                onClick={() => setStepIndex((index) => Math.min(STEPS.length - 1, index + 1))}
                className="sm:min-w-44"
              >
                Continue
              </Button>
            )}
          </div>
        </CardBody>
      </Card>

      <p className="mt-4 text-center text-[11px] text-ink-faint">
        In immediate danger? Call your local emergency number rather than waiting on this form.
      </p>
    </div>
  )
}

/* ------------------------------------------------------------------ step 2 */

/**
 * Where, and who could respond.
 *
 * The unit list is a *requirement* and the reason is stated when it is missing: a
 * case whose `unit_id` matches no unit is returned by no officer's and no unit
 * administrator's `GET /cases`. Offering "no preference" would be offering a
 * report that disappears, so the option does not exist.
 *
 * The pin is equally required — the contract defaults `latitude`/`longitude` to
 * zero, and a case pinned at 0,0 sends a unit to the Gulf of Guinea.
 */
function WhereStep({
  draft,
  onLocationChange,
  onPick,
  onClearPin,
  onHideLocationChange,
  onUnitChange,
  geoLoading,
  userLat,
  userLng,
  onUseCurrentLocation,
}: {
  draft: ReportDraft
  onLocationChange: (value: string) => void
  onPick: (lat: number, lng: number) => void
  onClearPin: () => void
  onHideLocationChange: (value: boolean) => void
  onUnitChange: (unitId: string) => void
  geoLoading: boolean
  userLat: number | null
  userLng: number | null
  onUseCurrentLocation: () => void
}) {
  const hasPin = draft.latitude !== null && draft.longitude !== null
  const nearby = useNearbyUnits(draft.latitude ?? undefined, draft.longitude ?? undefined)

  // The explicit null checks (rather than `hasPin`) are what narrow the pair for
  // the map's tuple prop — a boolean alias does not carry that information.
  const point: [number, number] | null =
    draft.latitude !== null && draft.longitude !== null
      ? [draft.latitude, draft.longitude]
      : null

  const inRange = nearby.data?.filter((unit) => unit.isInRange) ?? []
  const outOfRange = nearby.data?.filter((unit) => !unit.isInRange) ?? []

  return (
    <div className="flex flex-col gap-4">
      <MapView
        mode="pick"
        height="38vh"
        allowLocate
        label="Pick the location of the incident"
        pickLocation={point}
        onPickLocation={onPick}
      />

      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="flex items-center gap-1.5 text-xs text-ink-muted">
          <MapPin className="size-3.5 shrink-0" aria-hidden />
          {hasPin ? (
            <span className="tabular-nums">
              {formatCoord(draft.latitude ?? undefined, draft.longitude ?? undefined)}
            </span>
          ) : (
            'No pin placed yet'
          )}
        </p>
        {hasPin ? (
          <Button variant="ghost" size="sm" onClick={onClearPin}>
            Clear pin
          </Button>
        ) : null}
      </div>

      <Field
        label="Nearest landmark or street"
        hint="Optional, but it helps if the pin is slightly off."
      >
        {({ id, ...aria }) => (
          <Input
            id={id}
            {...aria}
            value={draft.location}
            maxLength={160}
            onChange={(event) => onLocationChange(event.target.value)}
            placeholder="Bode Thomas Road, by the filling station"
          />
        )}
      </Field>

      {(userLat != null && userLng != null) && (
        <Button
          variant="secondary"
          size="sm"
          className="w-full sm:w-auto"
          onClick={onUseCurrentLocation}
          disabled={geoLoading}
        >
          {geoLoading ? 'Locating…' : 'Use my current location'}
        </Button>
      )}

      {/* hideLocation: file the report against a coarse geohash instead of the
          precise coordinates. Matches the `POST /cases` field added alongside the
          location-sharing toggle. */}
      <label
        htmlFor="report-hide-location"
        className={cn(
          'flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors',
          draft.hideLocation
            ? 'border-warn/40 bg-warn/10'
            : 'border-border-hi bg-surface-hi',
        )}
      >
        <input
          id="report-hide-location"
          type="checkbox"
          checked={draft.hideLocation}
          onChange={(event) => onHideLocationChange(event.target.checked)}
          className="mt-0.5 size-4 shrink-0 accent-warn"
        />
        <span>
          <span className="block text-sm font-medium text-ink">
            File this report anonymously (hide my exact location)
          </span>
          <span className="mt-0.5 block text-xs text-ink-muted">
            Your report is kept against a coarse area instead of precise coordinates. Units still see the area to respond.
          </span>
        </span>
      </label>

      <fieldset className="flex flex-col gap-2">
        <legend className="text-xs font-medium text-ink-muted">
          Which unit should handle this?
        </legend>

        {!hasPin ? (
          <p className="rounded-lg border border-dashed border-border-hi px-3 py-6 text-center text-xs text-ink-faint">
            Place the pin first — we will list the units that cover that area.
          </p>
        ) : nearby.isLoading ? (
          <div className="flex flex-col gap-2">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-14 w-full" />
            ))}
          </div>
        ) : nearby.isError ? (
          <div className="rounded-lg border border-warn/30 bg-warn/10 px-3 py-3 text-xs text-warn">
            <p>Could not load the units covering this point. You can still try again.</p>
            <Button
              variant="ghost"
              size="sm"
              className="mt-2"
              onClick={() => void nearby.refetch()}
            >
              Try again
            </Button>
          </div>
        ) : !nearby.data || nearby.data.length === 0 ? (
          <p className="rounded-lg border border-dashed border-border-hi px-3 py-6 text-center text-xs text-ink-faint">
            No registered units were found near this point.
          </p>
        ) : (
          <>
            {inRange.length === 0 ? (
              <p className="text-[11px] text-ink-faint">
                No unit covers this point — these are the closest.
              </p>
            ) : null}
            {[...inRange, ...outOfRange].map((unit, index) => (
              <UnitOption
                key={unit.id}
                unit={unit}
                selected={draft.unitId === unit.id}
                closest={index === 0}
                onChange={() => onUnitChange(unit.id)}
              />
            ))}
          </>
        )}
      </fieldset>
    </div>
  )
}

function UnitOption({
  unit,
  selected,
  closest,
  onChange,
}: {
  unit: { id: string; name: string; city?: string; distance: number; isInRange: boolean }
  selected: boolean
  closest: boolean
  onChange: () => void
}) {
  return (
    <label
      className={cn(
        'flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors',
        selected ? 'border-signal/50 bg-signal/10' : 'border-border-hi bg-surface-hi',
      )}
    >
      <input
        type="radio"
        name="report-unit"
        checked={selected}
        onChange={onChange}
        className="mt-0.5 size-4 shrink-0 accent-signal"
      />
      <span className="min-w-0 flex-1">
        <span className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-ink">{unit.name}</span>
          {closest ? <Badge tone="signal">Closest</Badge> : null}
          {!unit.isInRange ? <Badge tone="warn">Outside usual coverage</Badge> : null}
        </span>
        <span className="mt-0.5 block text-xs text-ink-muted">
          <span className="tabular-nums">{unit.distance.toFixed(1)} km away</span>
          {unit.city ? ` · ${unit.city}` : ''}
        </span>
      </span>
    </label>
  )
}

/* ------------------------------------------------------------------ step 4 */

function ReviewStep({
  draft,
  photos,
  onEdit,
}: {
  draft: ReportDraft
  photos: PhotoFile[]
  onEdit: (index: number) => void
}) {
  // `useUnits`, not `useNearbyUnits`: this only needs a name for an id the
  // reporter already chose, so a pin-keyed query would be a second network call
  // for something already in hand. The unit list is cached for five minutes.
  const units = useUnits()
  const unitName = units.data?.find((unit) => unit.id === draft.unitId)?.name

  const links = draft.evidence.filter((link) => link.fileUrl.trim() !== '')

  return (
    <div className="flex flex-col gap-3">
      <SummaryRow label="What happened" onEdit={() => onEdit(0)}>
        <p className="text-sm font-medium text-ink">{draft.title}</p>
        <p className="mt-0.5 whitespace-pre-wrap text-sm text-ink-muted">{draft.description}</p>
        {draft.urgent ? (
          <span className="mt-2 inline-block">
            <Badge tone="warn">Marked urgent</Badge>
          </span>
        ) : null}
      </SummaryRow>

      <SummaryRow label="Where" onEdit={() => onEdit(1)}>
        <p className="flex items-center gap-1.5 text-sm text-ink">
          <MapPin className="size-3.5 shrink-0" aria-hidden />
          {draft.location.trim() || 'No landmark given'}
        </p>
        <p className="mt-0.5 tabular-nums text-xs text-ink-muted">
          {formatCoord(draft.latitude ?? undefined, draft.longitude ?? undefined)}
        </p>
        <p className="mt-1 text-xs text-ink-muted">
          {unitName ? `Unit asked to respond: ${unitName}` : 'Unit: none selected'}
        </p>
      </SummaryRow>

      <SummaryRow label="Evidence" onEdit={() => onEdit(2)}>
        {(links.length === 0 && photos.length === 0) ? (
          <p className="text-sm text-ink-muted">No links or photos</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {links.map((link, index) => (
              <li key={index} className="truncate text-xs text-ink-muted flex items-center gap-1">
                <Paperclip className="size-3" aria-hidden />
                {link.type} · {link.fileUrl}
              </li>
            ))}
            {photos.map((photo, index) => (
              <li key={index} className="truncate text-xs text-ink-muted flex items-center gap-1">
                <Image className="size-3" aria-hidden />
                {photo.file.name} ({Math.round(photo.file.size / 1024)} KB)
              </li>
            ))}
          </ul>
        )}
      </SummaryRow>

      <div className="rounded-lg border border-border-hi bg-surface-hi p-3 text-xs text-ink-muted">
        <p className="font-medium text-ink">What happens when you send this</p>
        <ul className="mt-1.5 flex list-disc flex-col gap-1 pl-4">
          <li>
            The report is recorded as <span className="text-ink">Pending</span> and waits for the
            unit to triage it. You can follow every status change on its tracking page.
          </li>
          <li>A tracking ID is issued the moment it is accepted — that is your reference.</li>
          <li>
            Reports are kept as public records, so do not include other people’s names, phone
            numbers or anything you would not want recorded.
          </li>
        </ul>
      </div>
    </div>
  )
}

function SummaryRow({
  label,
  onEdit,
  children,
}: {
  label: string
  onEdit: () => void
  children: ReactNode
}) {
  return (
    <div className="rounded-lg border border-border-hi p-3">
      <div className="mb-1.5 flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-ink-muted">{label}</span>
        <Button variant="ghost" size="sm" onClick={onEdit}>
          Edit
        </Button>
      </div>
      {children}
    </div>
  )
}

/* ---------------------------------------------------------------- receipt */

interface AttachmentState extends DraftEvidenceLink {
  status: 'queued' | 'sending' | 'done' | 'failed'
  message?: string
  /** For photos, the type (e.g., 'photo') and a preview URL */
  isPhoto?: boolean
  preview?: string
}

/**
 * The receipt — proof the report exists, and the tracking ID that makes it
 * followable.
 *
 * It renders from the create response, never from a refetched list, so a slow
 * refresh cannot make a filed report look unfiled. The evidence links are the one
 * thing still in flight here, and a failure is reported *beside the link that
 * failed*: the report itself already succeeded and must not be dressed up as a
 * partial failure.
 */
function Receipt({
  created,
  attachments,
  onCopy,
  onRetry,
  onStartAnother,
}: {
  created: CreateCaseResponse
  attachments: AttachmentState[]
  onCopy: (id: string) => void
  onRetry: (index: number) => void
  onStartAnother: () => void
}) {
  const navigate = useNavigate()
  const sendingIndex = attachments.findIndex((item) => item.status === 'sending')
  const failed = attachments.filter((item) => item.status === 'failed').length

  return (
    <div className="mx-auto w-full max-w-2xl p-4 sm:p-6">
      <Card>
        <CardBody className="flex flex-col items-center gap-4 py-8 text-center">
          <span className="grid size-12 place-items-center rounded-full bg-ok/10 text-ok">
            <CheckCircle2 className="size-6" aria-hidden />
          </span>

          <div>
            <h1 className="text-lg font-semibold text-ink">Report filed</h1>
            <p className="mt-1 text-sm text-ink-muted">
              It is with the unit for triage. Keep this reference — it is how you follow it.
            </p>
          </div>

          <div className="flex flex-wrap items-center justify-center gap-2">
            <StatusChip status={created.case.status} />
            <PriorityChip level={created.priorityLevel} />
          </div>

          <div className="w-full rounded-lg border border-border-hi bg-surface-hi p-3">
            <p className="text-[11px] text-ink-faint">Tracking ID</p>
            <div className="mt-1 flex items-center justify-center gap-2">
              <p className="tabular-nums text-lg font-semibold tracking-wide text-ink">
                {created.trackingId}
              </p>
              <Button
                variant="ghost"
                size="sm"
                aria-label="Copy tracking ID"
                onClick={() => onCopy(created.trackingId)}
              >
                <Copy className="size-4" aria-hidden />
              </Button>
            </div>
          </div>

          {attachments.length > 0 ? (
            <div className="w-full text-left">
              <p className="text-xs font-medium text-ink-muted">Your links</p>
              <ul className="mt-1.5 flex flex-col gap-1.5">
                {attachments.map((item, index) => (
                  <li
                    key={index}
                    className="flex items-start justify-between gap-2 rounded-lg border border-border-hi bg-surface-hi px-3 py-2"
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs text-ink">{item.fileUrl}</span>
                      <span
                        className={cn(
                          'mt-0.5 block text-[11px]',
                          item.status === 'failed' ? 'text-emergency' : 'text-ink-faint',
                        )}
                      >
                        {item.status === 'queued' ? 'Waiting to attach' : null}
                        {item.status === 'sending' ? 'Attaching…' : null}
                        {item.status === 'done' ? 'Attached to your report' : null}
                        {item.status === 'failed'
                          ? `Not attached — ${item.message ?? 'the upload failed'}`
                          : null}
                      </span>
                    </span>
                    {item.status === 'failed' ? (
                      <Button variant="secondary" size="sm" onClick={() => onRetry(index)}>
                        Retry
                      </Button>
                    ) : null}
                  </li>
                ))}
              </ul>
              {sendingIndex >= 0 ? (
                <p className="mt-2 text-[11px] text-ink-faint" aria-live="polite">
                  Attaching your links ({sendingIndex + 1} of {attachments.length})…
                </p>
              ) : failed > 0 ? (
                <p className="mt-2 text-[11px] text-ink-muted">
                  Your report is filed and the unit can see it. Only the links above did not get
                  through.
                </p>
              ) : null}
            </div>
          ) : null}

          <div className="mt-2 flex w-full flex-col gap-2 sm:flex-row sm:justify-center">
            {/* A Button, not a Link wrapping one: an anchor around a button is a
                nested interactive element — invalid markup, and it breaks the
                focus order for exactly the user who has to finish here. */}
            <Button
              variant="primary"
              block
              className="sm:w-auto"
              onClick={() => navigate(`/cases/${created.case.id}`)}
            >
              Track this report
            </Button>
            <Button variant="ghost" onClick={onStartAnother}>
              File another report
            </Button>
          </div>
        </CardBody>
      </Card>

      <p className="mt-4 text-center text-[11px] text-ink-faint">
        Your saved draft has been cleared.
      </p>
    </div>
  )
}
