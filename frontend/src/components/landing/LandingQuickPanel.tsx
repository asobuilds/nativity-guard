import { useEffect, useId, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { registerDialogLayer } from '@/components/ui/dialogLayer'

interface LandingQuickPanelProps {
  title: string
  onClose: () => void
  children: ReactNode
}

/** Public feature quick-view. Dialog layers share one keyboard/focus manager. */
export function LandingQuickPanel({ title, onClose, children }: LandingQuickPanelProps) {
  const dialogRef = useRef<HTMLElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const onCloseRef = useRef(onClose)
  onCloseRef.current = onClose
  const headingId = useId()

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return
    return registerDialogLayer(dialog, () => onCloseRef.current(), closeRef.current)
  }, [])

  return (
    <div className="ng-ref-modal-backdrop" onMouseDown={event => {
      if (event.currentTarget === event.target) onCloseRef.current()
    }}>
      <section
        ref={dialogRef}
        className="ng-ref-modal-window"
        role="dialog"
        aria-modal="true"
        aria-labelledby={headingId}
        tabIndex={-1}
      >
        <div className="ng-ref-modal-heading">
          <h2 id={headingId}>{title}</h2>
          <button ref={closeRef} type="button" onClick={() => onCloseRef.current()}
            aria-label={'Close ' + title}><X size={24} aria-hidden /></button>
        </div>
        <div className="ng-ref-modal-content">{children}</div>
      </section>
    </div>
  )
}
