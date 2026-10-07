import { describe, expect, it } from 'vitest'
import { freshResponderLocations, normaliseSosDetail, isSosActive, type SosDetail, type ResponderLocation } from './sosTracking'

const sos: SosDetail = { id: 'sos-1', userId: 'reporter', latitude: 0, longitude: 0, priority: 'high', status: 'dispatched', createdAt: '2026-10-06T12:00:00Z', responders: [] }
const now = Date.parse('2026-10-06T12:00:00Z')
const location: ResponderLocation = { id: 'response', unitId: 'unit', role: 'primary', latitude: 0, longitude: 0, accuracy: 12, recordedAt: new Date(now).toISOString() }

describe('SOS detail contract', () => {
  it('reads the backend alert envelope and the new sos envelope', () => {
    expect(normaliseSosDetail({ alert: sos })).toEqual(sos)
    expect(normaliseSosDetail({ sos, responders: [] })).toEqual(sos)
  })
  it('gives a deliberate error for a missing alert instead of dereferencing undefined', () => {
    expect(() => normaliseSosDetail({})).toThrow('did not return an SOS')
  })
  it('stops live views for either terminal status or terminal dispatch state', () => {
    expect(isSosActive(sos)).toBe(true)
    expect(isSosActive({ ...sos, status: 'cancelled' })).toBe(false)
    expect(isSosActive({ ...sos, status: 'resolved' })).toBe(false)
    expect(isSosActive({ ...sos, dispatchState: 'resolved' })).toBe(false)
  })
})

describe('live location freshness', () => {
  it('accepts valid zero coordinates, and expires data even without another fetch', () => {
    expect(freshResponderLocations([location], now)).toHaveLength(1)
    expect(freshResponderLocations([location], now + 120_001)).toEqual([])
  })
  it.each([
    { latitude: 91 }, { longitude: -181 }, { accuracy: -1 }, { latitude: NaN },
    { recordedAt: 'invalid' }, { recordedAt: new Date(now + 10_001).toISOString() },
  ])('excludes malformed or future positions: %j', (change) => {
    expect(freshResponderLocations([{ ...location, ...change }], now)).toEqual([])
  })
})
