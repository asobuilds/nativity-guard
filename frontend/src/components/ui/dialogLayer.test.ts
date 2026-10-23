import { describe, expect, it } from 'vitest'
import { LayerStack, wrappedTabTarget } from './dialogLayer'

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
