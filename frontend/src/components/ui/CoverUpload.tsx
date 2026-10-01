import { useRef, useState, type ChangeEvent } from 'react'
import { Image as ImageIcon, Loader2, Trash2, Upload } from 'lucide-react'
import { api, ApiError } from '@/lib/apiClient'

interface CoverUploadProps {
  currentUrl?: string
  onUploaded: () => void
  onDeleted: () => void
}

/**
 * Wide cover photo uploader, mirroring AvatarUpload for the rectangular
 * shape. Buttons float over the image, uploads go to POST /users/me/cover,
 * deletes to DELETE /users/me/cover.
 */
export function CoverUpload({ currentUrl, onUploaded, onDeleted }: CoverUploadProps) {
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

  return (
    <div className="relative h-40 w-full bg-gradient-to-r from-signal/25 via-signal/10 to-warn/20 sm:h-56">
      {currentUrl ? (
        <img src={currentUrl} alt="Cover" className="h-full w-full object-cover" />
      ) : (
        <div className="grid h-full w-full place-items-center text-ink-faint">
          <div className="flex flex-col items-center gap-2">
            <ImageIcon className="size-8" aria-hidden />
            <span className="text-xs">No cover photo yet</span>
          </div>
        </div>
      )}

      {uploading || deleting ? (
        <div className="absolute inset-0 grid place-items-center bg-black/40">
          <Loader2 className="size-6 animate-spin text-white" aria-hidden />
        </div>
      ) : null}

      <div className="absolute bottom-3 right-3 flex gap-2">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={handleFile}
        />
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading || deleting}
          className="inline-flex items-center gap-1.5 rounded-lg border border-white/20 bg-black/60 px-3 py-1.5 text-xs font-medium text-white backdrop-blur transition-colors hover:bg-black/80 disabled:opacity-50"
        >
          <Upload className="size-3.5" aria-hidden />
          {currentUrl ? 'Change cover' : 'Upload cover'}
        </button>
        {currentUrl ? (
          <button
            type="button"
            onClick={handleDelete}
            disabled={uploading || deleting}
            className="inline-flex items-center gap-1.5 rounded-lg border border-white/20 bg-black/60 px-3 py-1.5 text-xs font-medium text-white backdrop-blur transition-colors hover:bg-black/80 disabled:opacity-50"
          >
            <Trash2 className="size-3.5" aria-hidden />
            Remove
          </button>
        ) : null}
      </div>

      {error ? (
        <p className="absolute bottom-3 left-3 rounded-lg bg-black/70 px-3 py-1 text-xs text-warn">
          {error}
        </p>
      ) : null}
    </div>
  )
}