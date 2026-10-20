import { describe, expect, it } from 'vitest'
import {
  GEO_ERROR_DENIED,
  GEO_ERROR_TIMEOUT,
  GEO_ERROR_UNAVAILABLE,
  locateFailureKind,
  locateFailureMessage,
} from './locationNotice'

/**
 * The browser reports an insecure origin, a refusal, a timeout and a missing
 * feature very differently, and each needs a different instruction. These
 * cases pin the classification so the landing page never tells a visitor to
 * change a setting that cannot help them.
 */

describe('locateFailureKind', () => {
  it('returns null when nothing failed', () => {
    expect(locateFailureKind({ permission: 'granted', secureContext: true })).toBeNull()
  })

  it('reports an insecure origin before anything else', () => {
    // On http:// the browser reports code 1 (denied) whatever the visitor
    // answers, so the origin has to be checked first.
    expect(
      locateFailureKind({ errorCode: GEO_ERROR_DENIED, secureContext: false, supported: true }),
    ).toBe('insecure')
  })

  it('reports an unsupported browser', () => {
    expect(locateFailureKind({ supported: false, secureContext: true })).toBe('unsupported')
  })

  it('reports a refusal', () => {
    expect(
      locateFailureKind({ errorCode: GEO_ERROR_DENIED, secureContext: true, supported: true }),
    ).toBe('denied')
  })

  it('reports a timeout as a timeout, not a refusal', () => {
    expect(
      locateFailureKind({ errorCode: GEO_ERROR_TIMEOUT, secureContext: true, supported: true }),
    ).toBe('timeout')
  })

  it('reports an unavailable position as unavailable', () => {
    expect(
      locateFailureKind({
        errorCode: GEO_ERROR_UNAVAILABLE,
        secureContext: true,
        supported: true,
      }),
    ).toBe('unavailable')
  })

  it('falls back to the permission state when the browser gave no code', () => {
    expect(locateFailureKind({ permission: 'denied', secureContext: true })).toBe('denied')
    expect(locateFailureKind({ permission: 'unavailable', secureContext: true })).toBe('unsupported')
  })
})

describe('locateFailureMessage', () => {
  it('gives every kind its own instruction', () => {
    const kinds = ['insecure', 'denied', 'timeout', 'unavailable', 'unsupported'] as const
    const messages = kinds.map(locateFailureMessage)
    expect(new Set(messages).size).toBe(kinds.length)
    for (const message of messages) {
      expect(message.length).toBeGreaterThan(20)
    }
  })

  it('tells the visitor that https is the fix for an insecure origin', () => {
    expect(locateFailureMessage('insecure')).toMatch(/https/)
  })
})
