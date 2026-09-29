import { useEffect, useState } from 'react'
import { listQueued, removeQueued, updateQueued, QUEUE_CHANGED_EVENT, type QueuedWrite } from '@/lib/offlineQueue'
import { api, ApiError } from '@/lib/apiClient'

/**
 * Drains the offline write queue whenever we are (or come back) online.
 * Exposes the current pending count so a banner can show it.
 */
export function useOfflineQueue() {
  const [pending, setPending] = useState<QueuedWrite[]>([])
  const [draining, setDraining] = useState(false)
  const [justSent, setJustSent] = useState(0)

  async function refresh() {
    try {
      setPending(await listQueued())
    } catch {
      /* IDB unavailable — leave the list empty */
    }
  }

  useEffect(() => {
    void refresh()
    const onChange = () => void refresh()
    window.addEventListener(QUEUE_CHANGED_EVENT, onChange)
    return () => window.removeEventListener(QUEUE_CHANGED_EVENT, onChange)
  }, [])

  useEffect(() => {
    async function drain() {
      if (draining) return
      if (!navigator.onLine) return
      const items = await listQueued()
      if (items.length === 0) return
      setDraining(true)
      let sent = 0
      for (const entry of items) {
        try {
          await api.post(entry.path, entry.body)
          await removeQueued(entry.id)
          sent++
        } catch (err) {
          if (err instanceof ApiError && err.status === 0) {
            // still offline, stop trying
            break
          }
          if (err instanceof ApiError && err.status >= 400 && err.status < 500) {
            // validation error — the queued body will never succeed; drop it
            await removeQueued(entry.id)
            continue
          }
          // 5xx or other — bump attempts and leave it in the queue
          await updateQueued({ ...entry, attempts: entry.attempts + 1, lastError: String(err) })
        }
      }
      setDraining(false)
      if (sent > 0) {
        setJustSent(sent)
        window.setTimeout(() => setJustSent(0), 6000)
      }
      void refresh()
    }

    void drain()
    window.addEventListener('online', drain)
    return () => window.removeEventListener('online', drain)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draining])

  return { pending, count: pending.length, draining, justSent }
}