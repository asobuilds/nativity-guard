import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import { RotateCcw, X, ZoomIn, ZoomOut } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import {
  DEFAULT_COVER_ADJUST,
  coverImageStyle,
  type CoverAdjust,
} from '@/lib/coverPosition'

interface CoverEditorDialogProps {
  open: boolean
  imageUrl: string
  initial: CoverAdjust
  saving?: boolean
  onCancel: () => void
  onSave: (next: CoverAdjust) => void
}

/**
 * Full-screen cover adjuster.
 *
 * Drag inside the preview to pan the image; the zoom slider magnifies
 * around the current focal point. The live preview uses the exact same
 * style function as the pages that render the cover, so what you see
 * here is what you get everywhere.
 */
export function CoverEditorDialog({
  open,
  imageUrl,
  initial,
  saving = false,
  onCancel,
  onSave,
}: CoverEditorDialogProps) {
  const [adjust, setAdjust] = useState<CoverAdjust>(initial)
  const previewRef = useRef<HTMLDivElement | null>(null)
  const dragState = useRef<{ pointerId: number; startX: number; startY: number; startAdjust: CoverAdjust } | null>(null)

  // Reset to the caller's current values each time the dialog opens.
  useEffect(() => {
    if (open) setAdjust(initial)
  }, [open, initial])

  // Escape cancels.
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onCancel])

  if (!open) return null

  function onPointerDown(e: ReactPointerEvent<HTMLDivElement>) {
    if (!previewRef.current) return
    e.currentTarget.setPointerCapture(e.pointerId)
    dragState.current = {
      pointerId: e.pointerId,
      startX: e.clientX,
      startY: e.clientY,
      startAdjust: adjust,
    }
  }

  function onPointerMove(e: ReactPointerEvent<HTMLDivElement>) {
    const drag = dragState.current
    if (!drag || drag.pointerId !== e.pointerId || !previewRef.current) return
    const rect = previewRef.current.getBoundingClientRect()
    const dx = e.clientX - drag.startX
    const dy = e.clientY - drag.startY
    // Panning right shows more of the left of the image → decrease X.
    const nextX = clamp(drag.startAdjust.x - (dx / rect.width) * 100, 0, 100)
    const nextY = clamp(drag.startAdjust.y - (dy / rect.height) * 100, 0, 100)
    setAdjust((prev) => ({ ...prev, x: Math.round(nextX), y: Math.round(nextY) }))
  }

  function onPointerUp(e: ReactPointerEvent<HTMLDivElement>) {
    if (dragState.current?.pointerId === e.pointerId) {
      dragState.current = null
    }
  }

  function bumpZoom(delta: number) {
    setAdjust((prev) => ({ ...prev, zoom: clamp(prev.zoom + delta, 100, 300) }))
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Adjust cover photo"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) onCancel()
      }}
    >
      <div className="flex w-full max-w-2xl flex-col rounded-lg border border-border bg-base shadow-2xl">
        <header className="flex items-center justify-between border-b border-border p-4">
          <div>
            <h3 className="text-base font-semibold text-ink">Adjust cover photo</h3>
            <p className="text-xs text-ink-faint">
              Drag the image to reposition it. Use the slider to zoom.
            </p>
          </div>
          <button
            type="button"
            aria-label="Close"
            onClick={onCancel}
            className="grid size-8 place-items-center rounded-md text-ink-muted hover:bg-surface-hi"
          >
            <X className="size-4" aria-hidden />
          </button>
        </header>

        {/* Preview — same aspect as the real cover strip */}
        <div className="p-4">
          <div
            ref={previewRef}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerCancel={onPointerUp}
            className="relative h-56 w-full touch-none select-none overflow-hidden rounded-lg bg-surface-hi sm:h-72"
            style={{ cursor: dragState.current ? 'grabbing' : 'grab' }}
          >
            <img src={imageUrl} alt="" draggable={false} style={coverImageStyle(adjust)} />
          </div>

          {/* Zoom controls */}
          <div className="mt-4 flex items-center gap-3">
            <button
              type="button"
              onClick={() => bumpZoom(-10)}
              className="grid size-8 place-items-center rounded-md border border-border text-ink-muted hover:bg-surface-hi"
              aria-label="Zoom out"
            >
              <ZoomOut className="size-4" aria-hidden />
            </button>
            <input
              type="range"
              min={100}
              max={300}
              step={5}
              value={adjust.zoom}
              onChange={(e) =>
                setAdjust((prev) => ({ ...prev, zoom: Number(e.target.value) }))
              }
              className="flex-1 accent-signal"
              aria-label="Zoom"
            />
            <button
              type="button"
              onClick={() => bumpZoom(10)}
              className="grid size-8 place-items-center rounded-md border border-border text-ink-muted hover:bg-surface-hi"
              aria-label="Zoom in"
            >
              <ZoomIn className="size-4" aria-hidden />
            </button>
            <span className="w-12 text-right text-xs tabular text-ink-muted">
              {(adjust.zoom / 100).toFixed(1)}×
            </span>
          </div>

          <button
            type="button"
            onClick={() => setAdjust(DEFAULT_COVER_ADJUST)}
            className="mt-2 inline-flex items-center gap-1.5 text-xs text-ink-muted hover:text-ink"
          >
            <RotateCcw className="size-3.5" aria-hidden />
            Reset
          </button>
        </div>

        <footer className="flex items-center justify-end gap-2 border-t border-border p-4">
          <Button variant="ghost" onClick={onCancel} disabled={saving}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={() => onSave(adjust)}
            loading={saving}
            disabled={saving}
          >
            Save
          </Button>
        </footer>
      </div>
    </div>
  )
}

function clamp(v: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, v))
}