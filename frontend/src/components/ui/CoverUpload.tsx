import { useRef, useState, type ChangeEvent } from 'react'
import { Image as ImageIcon, Loader2, Sliders, Trash2 } from 'lucide-react'
import { api, ApiError } from '@/lib/apiClient'
import { coverImageStyle, type CoverAdjust } from '@/lib/coverPosition'
import { CoverEditorDialog } from '@/components/ui/CoverEditorDialog'

interface CoverUploadProps {
  currentUrl?: string
  adjust: CoverAdjust
  onUploaded: () => void
  onDeleted: () => void
  onAdjustSaved: (next: CoverAdjust) => void
  savingAdjust?: boolean
}

/**
 * Cover strip. Clicking anywhere opens the file picker when there is no
 * cover yet. Once a cover exists, clicking the image does nothing;
 * "Adjust" opens the editor dialog, and the trash icon removes it.
 *
 * Rendering uses `coverImageStyle` — the same style the editor previews,
 * so what users save is what every page shows.
 */
export function CoverUpload({
  currentUrl,
  adjust,
  onUploaded,
  onDeleted,
  onAdjustSaved,
  savingAdjust = false,
}: CoverUploadProps) {
  const [uploading, setUploading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [editorOpen, setEditorOpen] = useState(false)
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
      setEditorOpen(true)
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

  const busy = uploading || deleting || savingAdjust

  return (
    <>
      <div className="relative h-40 w-full overflow-hidden bg-gradient-to-r from-signal/25 via-signal/10 to-warn/20 sm:h-56">
        {currentUrl ? (
          <img
            src={currentUrl}
            alt=""
            draggable={false}
            style={coverImageStyle(adjust)}
            className="select-none"
          />
        ) : (
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={busy}
            aria-label="Add a cover photo"
            className="group absolute inset-0 grid place-items-center text-ink-faint focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
          >
            <div className="flex flex-col items-center gap-2">
              <ImageIcon className="size-8" aria-hidden />
              <span className="text-xs">Tap to add a cover photo</span>
            </div>
          </button>
        )}

        {busy && uploading ? (
          <div className="absolute inset-0 grid place-items-center bg-black/50">
            <Loader2 className="size-6 animate-spin text-white" aria-hidden />
          </div>
        ) : null}

        {currentUrl ? (
          <div className="absolute bottom-3 right-3 z-10 flex items-center gap-1.5">
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={busy}
              aria-label="Replace cover photo"
              className="rounded-lg border border-white/20 bg-black/60 px-3 py-1.5 text-xs font-medium text-white backdrop-blur transition-colors hover:bg-black/80 disabled:opacity-50"
            >
              Replace
            </button>
            <button
              type="button"
              onClick={() => setEditorOpen(true)}
              disabled={busy}
              aria-label="Adjust cover photo"
              className="inline-flex items-center gap-1.5 rounded-lg border border-white/20 bg-black/60 px-3 py-1.5 text-xs font-medium text-white backdrop-blur transition-colors hover:bg-black/80 disabled:opacity-50"
            >
              <Sliders className="size-3.5" aria-hidden />
              Adjust
            </button>
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

      {currentUrl ? (
        <CoverEditorDialog
          open={editorOpen}
          imageUrl={currentUrl}
          initial={adjust}
          saving={savingAdjust}
          onCancel={() => setEditorOpen(false)}
          onSave={(next) => {
            onAdjustSaved(next)
            setEditorOpen(false)
          }}
        />
      ) : null}
    </>
  )
}