import { useEffect, useState } from 'react'
import {
  BadgeCheck,
  Clock,
  FileText,
  Film,
  ImageOff,
  MapPin,
  Music,
  ShieldQuestion,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { downloadPrivateMedia } from '@/lib/apiClient'
import { formatCoord, formatDateTime, relativeTime } from '@/lib/format'
import { Badge } from '@/components/ui/Chips'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { EmptyState, Skeleton } from '@/components/ui/States'
import type { Evidence } from '@/types/api'

const TYPE_ICON: Record<string, typeof FileText> = {
  image: ImageOff,
  photo: ImageOff,
  video: Film,
  audio: Music,
  document: FileText,
}

function isImage(type: string | undefined): boolean {
  return type === 'image' || type === 'photo'
}


/** Load media through the authenticated case route only when the user previews it. */
function EvidenceMediaPreview({ evidence }: { evidence: Evidence }) {
  const [url, setUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const privateFile = evidence.fileUrl?.startsWith('evidence/')

  useEffect(() => {
    if (!privateFile) { setLoading(false); return }
    const abort = new AbortController()
    let objectURL: string | null = null
    setUrl(null)
    setError(null)
    setLoading(true)
    void downloadPrivateMedia(
      `/evidence/case/${encodeURIComponent(evidence.caseId)}/${encodeURIComponent(evidence.id)}/file`,
      abort.signal,
    ).then(blob => {
      if (abort.signal.aborted) return
      objectURL = URL.createObjectURL(blob)
      setUrl(objectURL)
    }).catch(err => {
      if (!abort.signal.aborted) setError(err instanceof Error ? err.message : 'Could not open this attachment.')
    }).finally(() => { if (!abort.signal.aborted) setLoading(false) })
    return () => {
      abort.abort()
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [privateFile, evidence.caseId, evidence.id])

  const source = privateFile ? url : /^https:\/\//i.test(evidence.fileUrl || '') ? evidence.fileUrl : null
  if (loading) return <p role="status" className="text-sm text-ink-muted">Loading private evidence…</p>
  if (error) return <p role="alert" className="text-sm text-emergency">{error}</p>
  if (!source) return <p className="text-sm text-ink-muted">This attachment cannot be previewed here.</p>
  if (isImage(evidence.type)) {
    return <img src={source} alt={evidence.description || 'Case evidence'} className="mx-auto max-h-[60vh] rounded-lg object-contain" referrerPolicy="no-referrer" />
  }
  if (evidence.type === 'video') {
    return <video controls playsInline src={source} className="mx-auto max-h-[60vh] w-full rounded-lg bg-black" />
  }
  if (evidence.type === 'audio') {
    return <audio controls src={source} className="w-full" />
  }
  return <a href={source} download="case-evidence" target="_blank" rel="noreferrer noopener" className="text-sm text-signal underline">Open attachment</a>
}

/**
 * Evidence gallery for a case. Verification is a separate, explicit action —
 * unverified items are never styled to look trustworthy.
 */
export function EvidenceGallery({
  evidence,
  isLoading = false,
  canVerify = false,
  onVerify,
  verifyingId,
  className,
}: {
  evidence: Evidence[]
  isLoading?: boolean
  /** Only the assigned officer / unit may verify. */
  canVerify?: boolean
  onVerify?: (id: string) => void
  verifyingId?: string | null
  className?: string
}) {
  const [preview, setPreview] = useState<Evidence | null>(null)

  if (isLoading) {
    return (
      <div className={cn('grid grid-cols-2 gap-3 p-4 sm:grid-cols-3', className)}>
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} className="aspect-4/3 w-full rounded-lg" />
        ))}
      </div>
    )
  }

  if (evidence.length === 0) {
    return (
      <EmptyState
        icon={<ShieldQuestion className="size-5" aria-hidden />}
        title="No evidence attached"
        description="Photos, recordings and documents attached by officers or the reporter will be listed here."
      />
    )
  }

  const verified = evidence.filter((e) => e.isVerified).length

  return (
    <div className={cn('p-4', className)}>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Badge tone="ok">
          <BadgeCheck className="size-3.5" aria-hidden />
          {verified} verified
        </Badge>
        <Badge>{evidence.length - verified} awaiting verification</Badge>
      </div>

      <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {evidence.map((item) => {
          const Icon = TYPE_ICON[item.type] ?? FileText
          return (
            <li
              key={item.id}
              className="flex flex-col overflow-hidden rounded-panel border border-border bg-surface"
            >
              <button
                type="button"
                onClick={() => setPreview(item)}
                className="group relative block aspect-4/3 w-full overflow-hidden bg-surface-hi text-left"
                aria-label={`Preview ${item.description || item.type} evidence`}
              >
                <span className="grid size-full place-items-center text-ink-faint">
                  <Icon className="size-7" aria-hidden />
                </span>
                <span className="absolute left-2 top-2">
                  {item.isVerified ? (
                    <Badge tone="ok" className="backdrop-blur">
                      <BadgeCheck className="size-3.5" aria-hidden />
                      Verified
                    </Badge>
                  ) : (
                    <Badge tone="warn" className="backdrop-blur">
                      <Clock className="size-3.5" aria-hidden />
                      Unverified
                    </Badge>
                  )}
                </span>
              </button>

              <div className="flex flex-1 flex-col gap-1.5 p-2.5">
                <p className="line-clamp-2 text-xs font-medium text-ink">
                  {item.description || `${item.type} evidence`}
                </p>
                <p className="flex items-center gap-1 text-[11px] text-ink-faint">
                  <MapPin className="size-3" aria-hidden />
                  {formatCoord(item.latitude, item.longitude)}
                </p>
                <time
                  dateTime={item.createdAt}
                  title={formatDateTime(item.createdAt)}
                  className="text-[11px] text-ink-faint"
                >
                  {relativeTime(item.createdAt)}
                </time>

                {canVerify && !item.isVerified && onVerify ? (
                  <Button
                    size="sm"
                    variant="secondary"
                    className="mt-1"
                    loading={verifyingId === item.id}
                    onClick={() => onVerify(item.id)}
                  >
                    Verify
                  </Button>
                ) : null}
              </div>
            </li>
          )
        })}
      </ul>

      <Modal
        open={Boolean(preview)}
        onClose={() => setPreview(null)}
        title={preview?.description || `${preview?.type ?? ''} evidence`}
        description={
          preview
            ? `Attached ${formatDateTime(preview.createdAt)} · ${formatCoord(preview.latitude, preview.longitude)}`
            : undefined
        }
        size="lg"
      >
        {preview ? <EvidenceMediaPreview evidence={preview} /> : null}
      </Modal>
    </div>
  )
}
