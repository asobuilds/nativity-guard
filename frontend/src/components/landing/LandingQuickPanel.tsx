import { useEffect, useId, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'

interface LandingQuickPanelProps {
  title: string
  onClose: () => void
  children: ReactNode
}

/** Accessible quick-view for existing public features; no API actions are sent
 * by opening or closing the dialog. Focus is contained and restored on close.
 */
export function LandingQuickPanel({ title, onClose, children }: LandingQuickPanelProps) {
  const dialogRef = useRef<HTMLElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const headingId = useId()

  useEffect(() => {
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    closeRef.current?.focus()

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        event.preventDefault()
        onClose()
        return
      }
      if (event.key !== 'Tab') return

      const focusable = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
      ) ?? []).filter(el => el.getAttribute('aria-hidden') !== 'true' && el.getClientRects().length > 0)
      if (!focusable.length) {
        event.preventDefault()
        closeRef.current?.focus()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && (document.activeElement === first || !dialogRef.current?.contains(document.activeElement))) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && (document.activeElement === last || !dialogRef.current?.contains(document.activeElement))) {
        event.preventDefault()
        first.focus()
      }
    }

    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      previousFocus?.focus()
    }
  }, [onClose])

  return (
    <div className="ng-ref-modal-backdrop" onMouseDown={event => {
      if (event.currentTarget === event.target) onClose()
    }}>
      <section ref={dialogRef} className="ng-ref-modal-window" role="dialog"
        aria-modal="true" aria-labelledby={headingId}>
        <div className="ng-ref-modal-heading">
          <h2 id={headingId}>{title}</h2>
          <button ref={closeRef} type="button" onClick={onClose}
            aria-label={'Close ' + title}><X size={24} aria-hidden /></button>
        </div>
        <div className="ng-ref-modal-content">{children}</div>
      </section>
    </div>
  )
}
