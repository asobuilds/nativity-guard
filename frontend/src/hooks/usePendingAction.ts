import { useCallback, useState } from 'react'
import type { ActionContext, StoredPendingAction } from '@/components/landing/types'

export const PENDING_ACTION_KEY = 'ng.pending_action'
export const MAX_AGE_MS = 2 * 60 * 60 * 1000 // 2 hours

export function readPendingAction(): StoredPendingAction | null {
  try {
    if (typeof window === 'undefined') return null
    const raw = sessionStorage.getItem(PENDING_ACTION_KEY) ?? localStorage.getItem(PENDING_ACTION_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredPendingAction
    if (!parsed || typeof parsed.savedAt !== 'number' || !parsed.context) {
      clearPendingAction()
      return null
    }
    if (Date.now() - parsed.savedAt > MAX_AGE_MS) {
      clearPendingAction()
      return null
    }
    return parsed
  } catch {
    return null
  }
}

export function writePendingAction(context: ActionContext): StoredPendingAction {
  const item: StoredPendingAction = {
    context,
    savedAt: Date.now(),
  }
  const serialized = JSON.stringify(item)
  try {
    if (typeof window !== 'undefined') {
      sessionStorage.setItem(PENDING_ACTION_KEY, serialized)
      localStorage.setItem(PENDING_ACTION_KEY, serialized)
    }
  } catch {
    /* private mode / storage full */
  }
  return item
}

export function clearPendingAction(): void {
  try {
    if (typeof window !== 'undefined') {
      sessionStorage.removeItem(PENDING_ACTION_KEY)
      localStorage.removeItem(PENDING_ACTION_KEY)
    }
  } catch {
    /* ignore */
  }
}

export function consumePendingActionStorage(): ActionContext | null {
  const item = readPendingAction()
  clearPendingAction()
  return item ? item.context : null
}

export function usePendingAction() {
  const [current, setCurrent] = useState<StoredPendingAction | null>(() => readPendingAction())

  const saveAction = useCallback((context: ActionContext) => {
    const saved = writePendingAction(context)
    setCurrent(saved)
    return saved
  }, [])

  const clearAction = useCallback(() => {
    clearPendingAction()
    setCurrent(null)
  }, [])

  const consumeAction = useCallback((): ActionContext | null => {
    const item = readPendingAction()
    clearPendingAction()
    setCurrent(null)
    return item ? item.context : null
  }, [])

  return {
    pendingAction: current?.context ?? null,
    pendingTimestamp: current?.savedAt ?? null,
    saveAction,
    clearAction,
    consumeAction,
  }
}
