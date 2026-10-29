import { describe, expect, it } from 'vitest'
import { ROLES, isKnownRole, normaliseRole } from './role'

/**
 * The role vocabulary is a contract the app cannot survive guessing at: an
 * unrecognised role used to reach `homePathForRole`'s fallback and redirect the app
 * to itself until React blanked the screen. These tests pin the literals the backend
 * stores and the two properties that make that impossible — a known role
 * normalises, and an unknown one returns `null` rather than a plausible stand-in.
 */

describe('the role vocabulary', () => {
  it('pins the literal strings the backend stores', () => {
    expect(ROLES).toEqual([
      'citizen',
      'officer',
      'unit_admin',
      'super_admin',
      'agency_division_admin',
      'agency_observer',
    ])
  })

  it('accepts each canonical role unchanged', () => {
    for (const role of ROLES) expect(normaliseRole(role)).toBe(role)
  })

  it('normalises the hyphenated spellings the database has carried', () => {
    expect(normaliseRole('super-admin')).toBe('super_admin')
    expect(normaliseRole('unit-admin')).toBe('unit_admin')
  })

  it('tolerates case and surrounding whitespace', () => {
    expect(normaliseRole('  SUPER-ADMIN ')).toBe('super_admin')
    expect(normaliseRole('Officer')).toBe('officer')
    expect(normaliseRole(' citizen ')).toBe('citizen')
  })

  it('returns null — never a guess — for anything it cannot name', () => {
    expect(normaliseRole('head_admin')).toBeNull()
    expect(normaliseRole('admin')).toBeNull()
    expect(normaliseRole('')).toBeNull()
    expect(normaliseRole(null)).toBeNull()
    expect(normaliseRole(undefined)).toBeNull()
  })

  it('never invents a role from a near-miss', () => {
    expect(normaliseRole('superadmin')).toBeNull()
    expect(normaliseRole('super__admin')).toBeNull()
  })
})

describe('the role type guard', () => {
  it('accepts only the canonical literals', () => {
    expect(isKnownRole('super_admin')).toBe(true)
    expect(isKnownRole('super-admin')).toBe(false)
    expect(isKnownRole(42)).toBe(false)
    expect(isKnownRole(null)).toBe(false)
    expect(isKnownRole(undefined)).toBe(false)
  })
})
