/**
 * Coordinates stacked modal dialogs. Only the topmost dialog may handle
 * Escape / Tab. Inactive dialogs are inert and removed from the accessibility
 * tree while a child dialog is open.
 *
 * Browser-only functions are used from effects, never during SSR rendering.
 */
export class LayerStack<T> {
  private items: T[] = []

  push(layer: T) {
    this.items.push(layer)
  }

  remove(layer: T) {
    const index = this.items.lastIndexOf(layer)
    if (index < 0) return false
    this.items.splice(index, 1)
    return true
  }

  top(): T | undefined {
    return this.items[this.items.length - 1]
  }

  entries(): readonly T[] {
    return this.items
  }

  get size() {
    return this.items.length
  }
}

/** A null result means normal browser Tab navigation may continue. */
export function wrappedTabTarget<T>(
  focusable: readonly T[],
  active: T | null,
  backwards: boolean,
): T | null {
  if (focusable.length === 0) return null
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (!focusable.includes(active as T)) return backwards ? last : first
  if (backwards && active === first) return last
  if (!backwards && active === last) return first
  return null
}

interface DialogLayer {
  root: HTMLElement
  close: () => void
  focusReturn: HTMLElement | null
}

const stack = new LayerStack<DialogLayer>()
let originalBodyOverflow: string | null = null

const FOCUSABLE_SELECTOR =
  'a[href],area[href],button:not([disabled]),input:not([disabled]),textarea:not([disabled]),select:not([disabled]),[tabindex]:not([tabindex="-1"])'

function getFocusable(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (element) =>
      element.getAttribute('aria-hidden') !== 'true' &&
      !element.hasAttribute('inert') &&
      element.getClientRects().length > 0,
  )
}

function syncLayers() {
  const top = stack.top()
  for (const layer of stack.entries()) {
    const isTop = layer === top
    layer.root.setAttribute('aria-modal', isTop ? 'true' : 'false')
    if (isTop) layer.root.removeAttribute('aria-hidden')
    else layer.root.setAttribute('aria-hidden', 'true')
    layer.root.inert = !isTop
  }
}

function onDocumentKeyDown(event: KeyboardEvent) {
  const layer = stack.top()
  if (!layer) return

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopImmediatePropagation()
    layer.close()
    return
  }
  if (event.key !== 'Tab') return

  const focusable = getFocusable(layer.root)
  if (focusable.length === 0) {
    event.preventDefault()
    layer.root.focus()
    return
  }

  const active = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null
  const target = wrappedTabTarget(focusable, active, event.shiftKey)
  if (target) {
    event.preventDefault()
    target.focus()
  }
}

/** Register on mount, unregister on unmount. Idempotent cleanup for Strict Mode. */
export function registerDialogLayer(
  root: HTMLElement,
  close: () => void,
  initialFocus?: HTMLElement | null,
): () => void {
  const focusReturn = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null

  if (stack.size === 0) {
    originalBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', onDocumentKeyDown)
  }

  const layer: DialogLayer = { root, close, focusReturn }
  stack.push(layer)
  syncLayers()
  ;(initialFocus ?? root).focus()

  return () => {
    const wasTop = stack.top() === layer
    if (!stack.remove(layer)) return
    root.inert = false
    root.removeAttribute('aria-hidden')
    root.removeAttribute('aria-modal')
    syncLayers()

    if (stack.size === 0) {
      document.removeEventListener('keydown', onDocumentKeyDown)
      document.body.style.overflow = originalBodyOverflow ?? ''
      originalBodyOverflow = null
    }

    if (wasTop) {
      if (focusReturn?.isConnected && !focusReturn.inert) {
        focusReturn.focus()
      } else {
        const nextTop = stack.top()?.root
        if (nextTop) (getFocusable(nextTop)[0] ?? nextTop).focus()
      }
    }
  }
}
