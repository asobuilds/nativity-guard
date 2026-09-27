import { useEffect, useState } from 'react'
import { User, Camera, Trash2, Loader2 } from 'lucide-react'
import { api, mediaURL } from '@/lib/apiClient'
import { useToast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'

interface AvatarUploadProps {
  currentUrl?: string
  size?: number
  onUploaded: (newPath: string) => void
  onDeleted?: () => void
  /** Initials (e.g. "AO") shown in place of the image when none is set or the
   *  image fails to load. Renders a neutral icon when omitted. */
  fallbackInitials?: string
  className?: string
}

const MAX_SIZE = 5 * 1024 * 1024 // 5 MB
const ACCEPTED_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']

export function AvatarUpload({
  currentUrl,
  size = 96,
  onUploaded,
  onDeleted,
  fallbackInitials,
  className,
}: AvatarUploadProps) {
  const { notify } = useToast()
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [imgError, setImgError] = useState(false)

  // A new url must get a fresh chance to load — a 404 from a previous path should
  // not keep masking the replacement.
  useEffect(() => {
    setImgError(false)
  }, [currentUrl])

  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    setError(null)

    if (!ACCEPTED_TYPES.includes(file.type)) {
      setError('Only JPEG, PNG, WebP, or GIF images are allowed.')
      event.target.value = ''
      return
    }
    if (file.size > MAX_SIZE) {
      setError('File must be 5 MB or smaller.')
      event.target.value = ''
      return
    }

    setUploading(true)
    try {
      const response = await api.upload<{ avatarPath: string }>('/users/me/avatar', file)
      onUploaded(response.avatarPath)
      notify('Avatar updated', 'success')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Upload failed'
      setError(message)
      notify(message, 'error')
    } finally {
      setUploading(false)
      event.target.value = ''
    }
  }

  const handleDelete = async () => {
    if (!onDeleted) return
    try {
      await api.delete('/users/me/avatar')
      onDeleted()
      notify('Avatar removed', 'success')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Could not remove avatar'
      notify(message, 'error')
    }
  }

  // `hasImage` tracks whether a url was supplied; `imgError` tracks whether the
  // browser failed to load it. When either says "no usable image", the fallback
  // (initials or icon) is shown instead — a 404 therefore never renders a broken
  // image glyph.
  const hasImage = Boolean(currentUrl)
  const showFallback = !hasImage || imgError
  const diameter = `${size}px`
  const displayUrl = mediaURL(currentUrl) ?? undefined

  return (
    <div className={cn('relative inline-flex flex-col items-center gap-2', className)}>
      <div className="relative">
        <label htmlFor="avatar-upload" className="cursor-pointer">
          {hasImage ? (
            <img
              src={displayUrl ?? ''}
              alt="Avatar"
              className={cn('rounded-full object-cover border border-border', diameter)}
              width={size}
              height={size}
              onError={(e) => {
                // Hide the broken image immediately, before React re-renders, so
                // the glyph never flashes. `setImgError` swaps the rendering to
                // the fallback on the next paint.
                e.currentTarget.style.display = 'none'
                setImgError(true)
              }}
            />
          ) : null}

          {/* Shown when there is no image, or when the image errored. Hidden
              (display:none via the `hidden` attribute) only while a valid image
              is on screen. */}
          <div
            className={cn(
              'rounded-full flex items-center justify-center bg-signal/10 text-signal border border-border',
              diameter,
            )}
            hidden={hasImage && !imgError}
          >
            {showFallback && fallbackInitials ? (
              <span className="text-2xl font-semibold" aria-hidden>
                {fallbackInitials}
              </span>
            ) : (
              <User
                className={cn('size-1/2', size >= 80 ? 'text-2xl' : 'text-xl')}
                aria-hidden
              />
            )}
          </div>
        </label>

        <input
          id="avatar-upload"
          type="file"
          accept="image/*"
          onChange={handleFileSelect}
          disabled={uploading}
          className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          aria-label="Upload avatar"
        />

        {!uploading && !hasImage && (
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <Camera className={cn('text-ink-faint', size >= 80 ? 'size-8' : 'size-6')} aria-hidden />
          </div>
        )}

        {uploading && (
          <div className="absolute inset-0 rounded-full flex items-center justify-center bg-black/50">
            <Loader2 className="size-6 text-white animate-spin" aria-hidden />
          </div>
        )}

        {onDeleted && hasImage && !imgError && !uploading && (
          <button
            type="button"
            onClick={handleDelete}
            className="absolute -top-1 -right-1 size-6 rounded-full flex items-center justify-center bg-emergency/90 text-white hover:bg-emergency focus:outline-none focus-visible:ring-2 focus-visible:ring-emergency"
            aria-label="Remove avatar"
          >
            <Trash2 className="size-3.5" aria-hidden />
          </button>
        )}
      </div>

      {error && (
        <p className="text-xs text-emergency text-center max-w-[160px]">{error}</p>
      )}
    </div>
  )
}
