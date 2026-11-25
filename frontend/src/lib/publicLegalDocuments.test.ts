import { describe, expect, it } from 'vitest'
import {
  formatLegalEffectiveDate,
  validatePublicLegalDocument,
  type PublicLegalKind,
} from './publicLegalDocuments'

function fixture(kind: PublicLegalKind) {
  return {
    document: {
      id: kind + '-1',
      kind,
      role: kind === 'privacy' ? 'all' : 'citizen',
      version: 'v1.0',
      title: kind === 'privacy' ? 'Privacy Notice' : 'Terms of Service',
      content: 'Example text for validation test only.',
      effectiveAt: '2026-10-23T00:00:00Z',
      isActive: true,
    },
    version: 'v1.0',
  }
}

describe('public legal document validation', () => {
  it('accepts the real document shape for both citizen terms and global privacy', () => {
    expect(validatePublicLegalDocument(fixture('terms'), 'terms').role).toBe('citizen')
    expect(validatePublicLegalDocument(fixture('privacy'), 'privacy').role).toBe('all')
  })

  it('rejects an inactive or empty document instead of showing placeholder terms', () => {
    expect(() => validatePublicLegalDocument({
      ...fixture('terms'),
      document: { ...fixture('terms').document, isActive: false },
    }, 'terms')).toThrow(/valid active/)
    expect(() => validatePublicLegalDocument({
      ...fixture('privacy'),
      document: { ...fixture('privacy').document, content: '' },
    }, 'privacy')).toThrow(/valid active/)
  })

  it('rejects the wrong kind or role, so privacy is never presented as Terms', () => {
    expect(() => validatePublicLegalDocument(fixture('privacy'), 'terms')).toThrow()
    expect(() => validatePublicLegalDocument({
      document: { ...fixture('terms').document, role: 'super_admin' },
    }, 'terms')).toThrow()
  })

  it('rejects malformed payloads, dates and missing versions', () => {
    expect(() => validatePublicLegalDocument(null, 'terms')).toThrow()
    expect(() => validatePublicLegalDocument({ document: [] }, 'privacy')).toThrow()
    expect(() => validatePublicLegalDocument({
      document: { ...fixture('terms').document, effectiveAt: 'bogus' },
    }, 'terms')).toThrow()
    expect(() => validatePublicLegalDocument({
      document: { ...fixture('privacy').document, version: '' },
    }, 'privacy')).toThrow()
  })

  it('renders a stable date without depending on the browser timezone', () => {
    expect(formatLegalEffectiveDate('2026-10-23T00:00:00Z')).toMatch(/2026/)
    expect(formatLegalEffectiveDate('not-a-date')).toBe('Date unavailable')
  })
})
