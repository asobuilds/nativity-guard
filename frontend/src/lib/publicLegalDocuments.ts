/**
 * Validation for public legal documents.
 *
 * A failed API call is not an invitation to display made-up legal terms.
 * Missing, inactive, or malformed documents must appear as unavailable.
 */
export type PublicLegalKind = 'terms' | 'privacy'

export interface PublicLegalDocument {
  id: string
  kind: PublicLegalKind
  role: string
  version: string
  title: string
  content: string
  effectiveAt: string
  isActive: boolean
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function validatePublicLegalDocument(
  payload: unknown,
  kind: PublicLegalKind,
): PublicLegalDocument {
  if (!isObject(payload) || !isObject(payload.document)) {
    throw new Error('The legal document response is invalid.')
  }

  const doc = payload.document
  const expectedRole = kind === 'privacy' ? 'all' : 'citizen'
  if (
    doc.kind !== kind ||
    doc.role !== expectedRole ||
    doc.isActive !== true ||
    typeof doc.id !== 'string' ||
    typeof doc.version !== 'string' ||
    doc.version.trim().length === 0 ||
    typeof doc.title !== 'string' ||
    doc.title.trim().length === 0 ||
    typeof doc.content !== 'string' ||
    doc.content.trim().length === 0 ||
    typeof doc.effectiveAt !== 'string' ||
    !Number.isFinite(Date.parse(doc.effectiveAt))
  ) {
    throw new Error('No valid active legal document is currently available.')
  }

  return {
    id: doc.id,
    kind,
    role: expectedRole,
    version: doc.version,
    title: doc.title,
    content: doc.content,
    effectiveAt: doc.effectiveAt,
    isActive: true,
  }
}

export function formatLegalEffectiveDate(date: string): string {
  const parsed = Date.parse(date)
  return Number.isFinite(parsed)
    ? new Date(parsed).toLocaleDateString(undefined, {
        day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC',
      })
    : 'Date unavailable'
}
