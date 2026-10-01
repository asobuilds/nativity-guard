import { useEffect, useRef, useState, type ChangeEvent, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle2, Clock, FileText, Loader2, ShieldCheck, Upload, XCircle } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { api, ApiError } from '@/lib/apiClient'
import { cn } from '@/lib/cn'

type Status = 'not_submitted' | 'pending' | 'verified' | 'rejected' | 'loading'

interface IdentityStatusResponse {
  status: 'not_submitted' | 'pending' | 'verified' | 'rejected'
  documentType?: string
  submittedAt?: string
  verifiedAt?: string
  rejectionReason?: string
}

interface UploadResponse {
  documentPath: string
  hash: string
  size: number
}

const DOC_TYPES = [
  { value: 'nin', label: 'National Identification Number (NIN)', hint: '11 digits' },
  { value: 'voters_card', label: "Voter's Card (PVC)", hint: 'Your VIN' },
  { value: 'drivers_license', label: "Driver's License", hint: 'Your license number' },
  { value: 'passport', label: 'International Passport', hint: 'Your passport number' },
]

export function VerifyIdentityPage() {
  const [status, setStatus] = useState<Status>('loading')
  const [existing, setExisting] = useState<IdentityStatusResponse | null>(null)
  const [docType, setDocType] = useState('nin')
  const [docNumber, setDocNumber] = useState('')
  const [documentPath, setDocumentPath] = useState<string | null>(null)
  const [uploadedName, setUploadedName] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .get<IdentityStatusResponse>('/identity/me')
      .then((res) => {
        if (cancelled) return
        setExisting(res)
        setStatus(res.status)
      })
      .catch(() => {
        if (cancelled) return
        setStatus('not_submitted')
      })
    return () => {
      cancelled = true
    }
  }, [])

  async function handleFileSelect(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setError(null)
    setUploading(true)
    try {
      const res = await api.upload<UploadResponse>('/identity/document', file)
      setDocumentPath(res.documentPath)
      setUploadedName(file.name)
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Upload failed. Try again.')
      setDocumentPath(null)
      setUploadedName(null)
      if (fileInputRef.current) fileInputRef.current.value = ''
    } finally {
      setUploading(false)
    }
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    const trimmed = docNumber.trim()
    if (!trimmed) {
      setError('Enter your document number.')
      return
    }
    if (docType === 'nin' && !/^\d{11}$/.test(trimmed)) {
      setError('A NIN is exactly 11 digits — no spaces, no letters.')
      return
    }
    if (!documentPath) {
      setError('Upload a photo or PDF of your document before submitting.')
      return
    }

    setSubmitting(true)
    try {
      await api.post('/identity/submit', {
        documentType: docType,
        documentNumber: trimmed,
        documentUrl: documentPath,
      })
      const res = await api.get<IdentityStatusResponse>('/identity/me')
      setExisting(res)
      setStatus(res.status)
      setDocNumber('')
      setDocumentPath(null)
      setUploadedName(null)
      if (fileInputRef.current) fileInputRef.current.value = ''
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Could not submit. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  if (status === 'loading') {
    return (
      <div className="mx-auto max-w-2xl p-6">
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-8 animate-spin text-signal" aria-hidden />
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl p-4 sm:p-6">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Verify your identity</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Verifying your government ID is optional but recommended. Officers and
          administrators will see that your identity has been checked, which
          strengthens trust in your reports.
        </p>
      </header>

      {status === 'verified' && existing ? (
        <Card className="border border-ok/30">
          <CardBody className="flex flex-col items-center gap-3 py-8 text-center">
            <CheckCircle2 className="size-12 text-ok" aria-hidden />
            <h2 className="text-lg font-semibold text-ink">Your identity is verified</h2>
            <p className="text-sm text-ink-muted">
              Verified{documentTypeLabel(existing.documentType)}
              {existing.verifiedAt
                ? ` on ${new Date(existing.verifiedAt).toLocaleDateString()}`
                : ''}
              .
            </p>
          </CardBody>
        </Card>
      ) : null}

      {status === 'pending' && existing ? (
        <Card className="border border-warn/30">
          <CardBody className="flex flex-col items-center gap-3 py-8 text-center">
            <Clock className="size-12 text-warn" aria-hidden />
            <h2 className="text-lg font-semibold text-ink">Verification in review</h2>
            <p className="text-sm text-ink-muted">
              Submitted{documentTypeLabel(existing.documentType)}
              {existing.submittedAt
                ? ` on ${new Date(existing.submittedAt).toLocaleDateString()}`
                : ''}
              . A platform administrator will review it shortly.
            </p>
          </CardBody>
        </Card>
      ) : null}

      {status === 'rejected' && existing ? (
        <Card className="border border-emergency/30">
          <CardBody className="flex flex-col gap-2 py-6">
            <div className="flex items-center gap-2">
              <XCircle className="size-6 text-emergency" aria-hidden />
              <h2 className="text-lg font-semibold text-ink">Verification was rejected</h2>
            </div>
            {existing.rejectionReason ? (
              <p className="text-sm text-ink-muted">
                <strong className="text-ink">Reason given:</strong> {existing.rejectionReason}
              </p>
            ) : null}
            <p className="text-sm text-ink-muted">
              You can submit again below with corrected details.
            </p>
          </CardBody>
        </Card>
      ) : null}

      {status === 'not_submitted' || status === 'rejected' ? (
        <Card className={status === 'rejected' ? 'mt-4' : ''}>
          <CardHeader
            title={status === 'rejected' ? 'Resubmit your identity' : 'Submit your identity'}
          />
          <CardBody>
            <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
              <Field label="Document type" required>
                {({ id, ...aria }) => (
                  <select
                    id={id}
                    {...aria}
                    value={docType}
                    onChange={(e) => setDocType(e.target.value)}
                    className="h-10 w-full rounded-lg border border-border bg-surface px-3 text-sm text-ink"
                  >
                    {DOC_TYPES.map((d) => (
                      <option key={d.value} value={d.value}>
                        {d.label}
                      </option>
                    ))}
                  </select>
                )}
              </Field>

              <Field
                label="Document number"
                required
                hint={DOC_TYPES.find((d) => d.value === docType)?.hint}
              >
                {({ id, ...aria }) => (
                  <Input
                    id={id}
                    {...aria}
                    value={docNumber}
                    onChange={(e) =>
                      setDocNumber(
                        docType === 'nin'
                          ? e.target.value.replace(/\D/g, '').slice(0, 11)
                          : e.target.value,
                      )
                    }
                    autoComplete="off"
                  />
                )}
              </Field>

              <div>
                <label className="mb-1 block text-sm font-medium text-ink">
                  Document photo <span className="text-emergency">*</span>
                </label>
                <p className="mb-2 text-xs text-ink-faint">
                  A clear photo or scan of your ID. PDF, JPEG or PNG · up to 10 MB.
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".pdf,.jpg,.jpeg,.png"
                  className="hidden"
                  onChange={handleFileSelect}
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploading || submitting}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-lg border border-dashed border-border bg-surface-hi/30 px-4 py-3 text-left transition-colors hover:bg-surface-hi/50 disabled:opacity-60',
                  )}
                >
                  {uploading ? (
                    <Loader2 className="size-5 shrink-0 animate-spin text-signal" aria-hidden />
                  ) : documentPath ? (
                    <FileText className="size-5 shrink-0 text-ok" aria-hidden />
                  ) : (
                    <Upload className="size-5 shrink-0 text-ink-muted" aria-hidden />
                  )}
                  <span className="min-w-0 flex-1 text-sm">
                    {uploading
                      ? 'Uploading…'
                      : documentPath
                        ? `Attached: ${uploadedName}`
                        : 'Click to choose a file'}
                  </span>
                </button>
              </div>

              <div className="flex items-start gap-2 rounded-lg border border-border bg-surface-hi p-3 text-xs text-ink-muted">
                <ShieldCheck className="mt-0.5 size-4 shrink-0 text-signal" aria-hidden />
                <span>
                  Your document is stored in private storage. It is never publicly
                  reachable and only a platform administrator can view it during
                  the review. Your ID number is hashed — the raw value never
                  reaches our database.
                </span>
              </div>

              {error ? (
                <p role="alert" className="text-sm text-emergency">
                  {error}
                </p>
              ) : null}

              <div className="flex items-center gap-2">
                <Button
                  type="submit"
                  variant="primary"
                  loading={submitting}
                  disabled={submitting || uploading}
                >
                  Submit for review
                </Button>
                <Link
                  to="/profile"
                  className="inline-flex items-center px-3 text-sm text-ink-muted hover:text-ink"
                >
                  Cancel
                </Link>
              </div>
            </form>
          </CardBody>
        </Card>
      ) : null}
    </div>
  )
}

function documentTypeLabel(type?: string): string {
  if (!type) return ''
  const map: Record<string, string> = {
    nin: ' with your National Identification Number',
    voters_card: " with your Voter's Card",
    drivers_license: " with your Driver's License",
    passport: ' with your International Passport',
  }
  return map[type] ?? ''
}