import { useRef, useState, type ChangeEvent } from 'react'
import { Image as ImageIcon, Loader2, Trash2 } from 'lucide-react'
import { api, ApiError } from '@/lib/apiClient'
import { cn } from '@/lib/cn'
import { coverObjectPosition, type CoverPosition } from '@/lib/coverPosition'

interface CoverUploadProps {
  currentUrl?: string
  position: CoverPosition
  onUploaded: () => void
  onDeleted: () => void
  onPositionChange: (position: CoverPosition) => void
  savingPosition?: boolean
}

/**
 * Wide cover-photo uploader.
 *
 * The whole image area is one click target — clicking anywhere opens the
 * file picker. The position and remove controls float on top and stop
 * propagation so they don't trigger the upload.
 *
 * The image uses `object-position` derived from the `position` prop so a
 * portrait photo shows the face by default (`top`) instead of the chest.
 */
export function CoverUpload({
  currentUrl,
  position,
  onUploaded,
  onDeleted,
  onPositionChange,
  savingPosition = false,
}: CoverUploadProps) {
  const [uploading, setUploading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return
    setUploading(true)
    setError(null)
    try {
      await api.upload<unknown>('/users/me/cover', file)
      onUploaded()
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Upload failed')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  async function handleDelete() {
    setDeleting(true)
    setError(null)
    try {
      await api.delete('/users/me/cover')
      onDeleted()
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Delete failed')
    } finally {
      setDeleting(false)
    }
  }

  const busy = uploading || deleting || savingPosition

  return (
    <div className="relative h-40 w-full overflow-hidden bg-gradient-to-r from-signal/25 via-signal/10 to-warn/20 sm:h-56">
      {/* Whole area is the upload target */}
      <button
        type="button"
        onClick={() => fileInputRef.current?.click()}
        disabled={busy}
        aria-label={currentUrl ? 'Change cover photo' : 'Add a cover photo'}
        className="group absolute inset-0 block h-full w-full cursor-pointer text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
      >
        {currentUrl ? (
          <img
            src={currentUrl}
            alt=""
            className="h-full w-full object-cover"
            style={{ objectPosition: coverObjectPosition(position) }}
          />
        ) : (
          <div className="grid h-full w-full place-items-center text-ink-faint">
            <div className="flex flex-col items-center gap-2">
              <ImageIcon className="size-8" aria-hidden />
              <span className="text-xs">Tap to add a cover photo</span>
            </div>
          </div>
        )}

        <span className="pointer-events-none absolute inset-0 hidden items-center justify-center bg-black/0 transition-colors group-hover:bg-black/30 sm:flex">
          <span className="rounded-lg bg-black/70 px-3 py-1.5 text-xs font-medium text-white opacity-0 transition-opacity group-hover:opacity-100">
            {currentUrl ? 'Click to change cover' : 'Click to upload cover'}
          </span>
        </span>

        {busy ? (
          <span className="absolute inset-0 grid place-items-center bg-black/50">
            <Loader2 className="size-6 animate-spin text-white" aria-hidden />
          </span>
        ) : null}
      </button>

      {/* Position + Remove — only when there is a cover to adjust */}
      {currentUrl ? (
        <div className="absolute bottom-3 right-3 z-10 flex items-center gap-1.5">
          <div className="flex overflow-hidden rounded-lg border border-white/20 bg-black/60 backdrop-blur">
            {(['top', 'center', 'bottom'] as const).map((p) => (
              <button
                key={p}
                type="button"
                onClick={(event) => {
                  event.stopPropagation()
                  if (position !== p) onPositionChange(p)
                }}
                disabled={busy}
                aria-label={`Show the ${p} of the cover`}
                className={cn(
                  'px-2.5 py-1 text-[10px] font-medium transition-colors',
                  position === p
                    ? 'bg-white/25 text-white'
                    : 'text-white/70 hover:bg-white/10',
                )}
              >
                {p === 'top' ? 'Top' : p === 'center' ? 'Mid' : 'Bot'}
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation()
              void handleDelete()
            }}
            disabled={busy}
            aria-label="Remove cover photo"
            className="rounded-lg border border-white/20 bg-black/60 p-1.5 text-white backdrop-blur transition-colors hover:bg-black/80 disabled:opacity-50"
          >
            <Trash2 className="size-3.5" aria-hidden />
          </button>
        </div>
      ) : null}

      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={handleFile}
      />

      {error ? (
        <p className="absolute bottom-3 left-3 z-10 rounded-lg bg-black/70 px-3 py-1 text-xs text-warn">
          {error}
        </p>
      ) : null}
    </div>
  )
}