import { useEffect, useState } from 'react'
import { Film, Music, Paperclip } from 'lucide-react'
import { downloadPrivateMedia } from '@/lib/apiClient'
import type { SosMediaAttachment } from '@/lib/sosTracking'

export function SosAttachmentPanel({ sosId, media }: { sosId: string; media: SosMediaAttachment[] }) {
  const [activeId, setActiveId] = useState<string | null>(null)
  const [url, setUrl] = useState<string | null>(null)
  const [contentType, setContentType] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!activeId) return
    const abort = new AbortController()
    let objectUrl: string | null = null
    setUrl(null)
    setError('')
    setLoading(true)
    void downloadPrivateMedia(
      `/sos/${encodeURIComponent(sosId)}/media/${encodeURIComponent(activeId)}/file`,
      abort.signal,
    ).then(blob => {
      if (abort.signal.aborted) return
      objectUrl = URL.createObjectURL(blob)
      setContentType(blob.type)
      setUrl(objectUrl)
    }).catch(err => {
      if (!abort.signal.aborted) setError(err instanceof Error ? err.message : 'Unable to open this recording.')
    }).finally(() => { if (!abort.signal.aborted) setLoading(false) })
    return () => {
      abort.abort()
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [activeId, sosId])

  if (!media.length) return null
  return (
    <section aria-labelledby="sos-media-heading" className="rounded-panel border border-border bg-surface p-5">
      <h2 id="sos-media-heading" className="mb-3 text-sm font-semibold text-ink">
        Attached recordings ({media.length})
      </h2>
      <div className="flex flex-wrap gap-2">
        {media.map((item, index) => (
          <button
            key={item.id}
            type="button"
            aria-pressed={activeId === item.id}
            onClick={() => setActiveId(old => old === item.id ? null : item.id)}
            className="flex min-h-11 items-center gap-2 rounded-lg border border-border-hi bg-surface-hi px-3 text-sm text-ink focus-visible:outline-2 focus-visible:outline-signal"
          >
            {item.kind === 'audio' ? <Music className="size-4" /> : item.kind === 'video' ? <Film className="size-4" /> : <Paperclip className="size-4" />}
            Recording {index + 1}
          </button>
        ))}
      </div>
      {activeId && (
        <div className="mt-4 rounded-lg border border-border bg-surface-hi p-3">
          {loading ? <p role="status" className="text-sm text-ink-muted">Opening private recording…</p> : null}
          {error ? <p role="alert" className="text-sm text-emergency">{error}</p> : null}
          {url && (contentType.startsWith('audio/') ? (
            <audio controls src={url} className="w-full" />
          ) : contentType.startsWith('video/') ? (
            <video controls playsInline src={url} className="max-h-72 w-full rounded-lg bg-black" />
          ) : (
            <a href={url} download="sos-attachment" className="text-sm text-signal underline">Download private recording</a>
          ))}
        </div>
      )}
    </section>
  )
}
