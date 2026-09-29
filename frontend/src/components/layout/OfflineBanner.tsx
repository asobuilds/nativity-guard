import { useEffect, useState } from 'react'
import { Cloud, CloudOff, Check } from 'lucide-react'
import { useOfflineQueue } from '@/hooks/useOfflineQueue'

export function OfflineBanner() {
  const [offline, setOffline] = useState(!navigator.onLine)
  const { count, draining, justSent } = useOfflineQueue()

  useEffect(() => {
    const on = () => setOffline(false)
    const off = () => setOffline(true)
    window.addEventListener('online', on)
    window.addEventListener('offline', off)
    return () => {
      window.removeEventListener('online', on)
      window.removeEventListener('offline', off)
    }
  }, [])

  // Nothing to show if we're online, the queue is empty, and nothing just sent.
  if (!offline && count === 0 && justSent === 0) return null

  if (justSent > 0) {
    return (
      <div role="status" className="fixed inset-x-0 top-0 z-[70] flex items-center justify-center gap-2 bg-ok/90 px-4 py-2 text-sm text-white">
        <Check className="size-4" aria-hidden />
        {justSent} queued {justSent === 1 ? 'item' : 'items'} sent
      </div>
    )
  }

  if (offline) {
    return (
      <div role="status" className="fixed inset-x-0 top-0 z-[70] flex items-center justify-center gap-2 bg-warn px-4 py-2 text-sm text-signal-ink">
        <CloudOff className="size-4" aria-hidden />
        Offline — {count > 0 ? count + ' waiting to send' : 'reports will be saved locally'}
      </div>
    )
  }

  if (count > 0) {
    return (
      <div role="status" className="fixed inset-x-0 top-0 z-[70] flex items-center justify-center gap-2 bg-signal px-4 py-2 text-sm text-signal-ink">
        <Cloud className="size-4" aria-hidden />
        {draining ? 'Sending…' : count + ' waiting to send'}
      </div>
    )
  }

  return null
}