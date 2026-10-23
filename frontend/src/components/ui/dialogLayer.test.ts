import { describe, expect, it, vi } from 'vitest'
import { LayerStack, registerDialogLayer, wrappedTabTarget } from './dialogLayer'

describe('dialog focus layer ordering', () => {
  it('routes top-layer handling to the newest open dialog', () => {
    const stack = new LayerStack<string>()
    stack.push('feature-quick-view')
    expect(stack.top()).toBe('feature-quick-view')
    stack.push('donation-dialog')
    expect(stack.top()).toBe('donation-dialog')
    expect(stack.entries()).toEqual(['feature-quick-view', 'donation-dialog'])
    expect(stack.remove('donation-dialog')).toBe(true)
    expect(stack.top()).toBe('feature-quick-view')
  })

  it('does not accidentally pop the child when an older layer is removed', () => {
    const stack = new LayerStack<string>()
    stack.push('parent')
    stack.push('child')
    expect(stack.remove('parent')).toBe(true)
    expect(stack.top()).toBe('child')
    expect(stack.remove('missing')).toBe(false)
    expect(stack.size).toBe(1)
    expect(stack.remove('child')).toBe(true)
    expect(stack.top()).toBeUndefined()
    expect(stack.size).toBe(0)
  })
})

describe('keyboard navigation within active dialog', () => {
  const items = ['close', 'first-field', 'submit']

  it('wraps forward Tab from last to first', () => {
    expect(wrappedTabTarget(items, 'submit', false)).toBe('close')
  })

  it('wraps Shift+Tab from first to last', () => {
    expect(wrappedTabTarget(items, 'close', true)).toBe('submit')
  })

  it('moves outside focus back into the active dialog', () => {
    expect(wrappedTabTarget(items, 'background-link', false)).toBe('close')
    expect(wrappedTabTarget(items, 'background-link', true)).toBe('submit')
  })

  it('leaves normal navigation alone for interior focus and no controls', () => {
    expect(wrappedTabTarget(items, 'first-field', false)).toBeNull()
    expect(wrappedTabTarget(items, 'first-field', true)).toBeNull()
    expect(wrappedTabTarget([], null, false)).toBeNull()
  })
})

describe('nested Escape and focus behavior (simulated DOM)', () => {
  it('closes only the child, restores parent focus, and unlocks scrolling last', () => {
    const events = new Map<string, (event: KeyboardEvent) => void>()
    let active: FakeElement | null = null
    const setActiveElement = (element: FakeElement) => { active = element }

    class FakeElement {
      inert = false
      isConnected = true
      children: FakeElement[] = []
      attrs = new Map<string, string>()

      focus() { setActiveElement(this) }
      setAttribute(name: string, value: string) { this.attrs.set(name, value) }
      removeAttribute(name: string) { this.attrs.delete(name) }
      getAttribute(name: string) { return this.attrs.get(name) ?? null }
      hasAttribute(name: string) { return this.attrs.has(name) }
      getClientRects() { return [1] }
      querySelectorAll() { return this.children }
    }

    const doc = {
      body: { style: { overflow: 'auto' } },
      get activeElement() { return active },
      addEventListener: (name: string, handler: (event: KeyboardEvent) => void) => {
        events.set(name, handler)
      },
      removeEventListener: (name: string) => { events.delete(name) },
    }
    vi.stubGlobal('HTMLElement', FakeElement)
    vi.stubGlobal('document', doc)

    const pageButton = new FakeElement()
    const parent = new FakeElement()
    const parentClose = new FakeElement()
    parent.children = [parentClose]
    const child = new FakeElement()
    const childClose = new FakeElement()
    child.children = [childClose]
    let parentClosed = 0
    let childClosed = 0
    let unregisterParent: (() => void) | null = null
    let unregisterChild: (() => void) | null = null

    try {
      pageButton.focus()
      unregisterParent = registerDialogLayer(parent as unknown as HTMLElement, () => { parentClosed++ }, parentClose as unknown as HTMLElement)
      expect(active).toBe(parentClose)
      unregisterChild = registerDialogLayer(child as unknown as HTMLElement, () => { childClosed++ }, childClose as unknown as HTMLElement)
      expect(parent.inert).toBe(true)
      expect(parent.getAttribute('aria-hidden')).toBe('true')
      expect(active).toBe(childClose)
      expect(doc.body.style.overflow).toBe('hidden')

      const keyEvent = {
        key: 'Escape',
        preventDefault: vi.fn(),
        stopImmediatePropagation: vi.fn(),
      } as unknown as KeyboardEvent
      events.get('keydown')?.(keyEvent)
      expect(childClosed).toBe(1)
      expect(parentClosed).toBe(0)
      expect(keyEvent.preventDefault).toHaveBeenCalled()

      unregisterChild()
      unregisterChild = null
      expect(parent.inert).toBe(false)
      expect(parent.getAttribute('aria-modal')).toBe('true')
      expect(active).toBe(parentClose)
      expect(doc.body.style.overflow).toBe('hidden')

      unregisterParent()
      unregisterParent = null
      expect(doc.body.style.overflow).toBe('auto')
      expect(active).toBe(pageButton)
      expect(events.size).toBe(0)
    } finally {
      unregisterChild?.()
      unregisterParent?.()
      vi.unstubAllGlobals()
    }
  })
})
