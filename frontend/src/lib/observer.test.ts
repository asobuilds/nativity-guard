import { describe, expect, it } from 'vitest'
import {
  observerCategoryLabel,
  observerLocationLabel,
  observerStatusLabel,
  observerStatusTone,
} from './observer'

describe('observer dashboard labels', () => {
  it('labels known incident categories and falls back to the raw value', () => {
    expect(observerCategoryLabel('road_accident')).toBe('Road accident')
    expect(observerCategoryLabel('uncategorized')).toBe('Uncategorised')
    expect(observerCategoryLabel('')).toBe('Uncategorised')
    expect(observerCategoryLabel('custom')).toBe('custom')
  })

  it('humanises backend statuses', () => {
    expect(observerStatusLabel('on_scene')).toBe('On Scene')
    expect(observerStatusLabel('pending_admin_review')).toBe('Pending Admin Review')
    expect(observerStatusLabel('')).toBe('Unknown')
  })

  it('tones resolution, escalation, dispatch, and open states distinctly', () => {
    expect(observerStatusTone('closed')).toBe('ok')
    expect(observerStatusTone('escalated')).toBe('emergency')
    expect(observerStatusTone('dispatched')).toBe('warn')
    expect(observerStatusTone('pending')).toBe('signal')
    expect(observerStatusTone('unknown')).toBe('neutral')
  })

  it('formats the widest safe location first without inventing data', () => {
    expect(observerLocationLabel({ state: 'Kaduna', lga: 'Zaria', ward: 'Tudun Wada' })).toBe(
      'Tudun Wada, Zaria, Kaduna',
    )
    expect(observerLocationLabel({ state: 'Kaduna', lga: 'Zaria' })).toBe('Zaria, Kaduna')
    expect(observerLocationLabel({ state: '', lga: '' })).toBe('Location withheld')
  })
})
