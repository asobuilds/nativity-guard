import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import {
  readPendingAction,
  writePendingAction,
  clearPendingAction,
  consumePendingActionStorage,
  PENDING_ACTION_KEY,
  MAX_AGE_MS,
} from './usePendingAction'
import type { ActionContext } from '@/components/landing/types'

describe('usePendingAction storage helpers', () => {
  const store = new Map<string, string>()

  beforeEach(() => {
    store.clear()
    const mockStorage = {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, val: string) => store.set(key, val),
      removeItem: (key: string) => store.delete(key),
      clear: () => store.clear(),
      length: 0,
      key: () => null,
    }
    vi.stubGlobal('sessionStorage', mockStorage)
    vi.stubGlobal('localStorage', mockStorage)
    vi.stubGlobal('window', {})
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns null when storage is empty', () => {
    expect(readPendingAction()).toBeNull()
  })

  it('writes and reads a pending action', () => {
    const action: ActionContext = {
      type: 'report_incident',
      title: 'Report Incident at Kubwa',
      description: 'Sign in to file report',
      returnUrl: '/report',
      payload: { lat: 9.15, lng: 7.33 },
    }

    const written = writePendingAction(action)
    expect(written.context).toEqual(action)
    expect(typeof written.savedAt).toBe('number')

    const read = readPendingAction()
    expect(read?.context).toEqual(action)
  })

  it('clears pending action', () => {
    const action: ActionContext = {
      type: 'sos_broadcast',
      title: 'Emergency SOS',
      description: 'Sign in to broadcast SOS',
    }

    writePendingAction(action)
    expect(readPendingAction()).not.toBeNull()

    clearPendingAction()
    expect(readPendingAction()).toBeNull()
  })

  it('consumes action atomically', () => {
    const action: ActionContext = {
      type: 'join_unit',
      title: 'Join Bwari Guard',
      description: 'Sign in to apply',
    }

    writePendingAction(action)
    const consumed = consumePendingActionStorage()
    expect(consumed).toEqual(action)
    expect(readPendingAction()).toBeNull()
  })

  it('discards expired pending actions', () => {
    const action: ActionContext = {
      type: 'community_post',
      title: 'Post in LGA',
      description: 'Sign in to post',
    }

    const expiredItem = {
      context: action,
      savedAt: Date.now() - (MAX_AGE_MS + 1000),
    }
    store.set(PENDING_ACTION_KEY, JSON.stringify(expiredItem))

    expect(readPendingAction()).toBeNull()
  })
})
