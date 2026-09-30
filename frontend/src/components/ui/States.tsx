import type { ReactNode } from 'react'
import { AlertTriangle, Inbox, Loader2, WifiOff } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Button } from './Button'

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn('size-5 animate-spin text-ink-muted', className)} aria-hidden />
}

export function FullPageSpinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div
      role="status"
      aria-live="polite"
      className="flex min-h-screen flex-col items-center justify-center gap-3 bg-base text-ink-muted"
    >
      <Loader2 className="size-6 animate-spin" aria-hidden />
      <span className="text-sm">{label}</span>
    </div>
  )
}

/** Shimmering block used while a data surface loads. */
export function Skeleton({ className }: { className?: string }) {
  return (
    <div
      aria-hidden
      className={cn('ng-skeleton rounded-md bg-surface-hi', className)}
    />
  )
}

export function EmptyState({
  title,
  description,
  icon,
  action,
}: {
  title: string
  description?: string
  icon?: ReactNode
  action?: ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-14 text-center">
      <div className="grid size-12 place-items-center rounded-full bg-surface-hi text-ink-faint">
        {icon ?? <Inbox className="size-5" aria-hidden />}
      </div>
      <div>
        <p className="text-sm font-semibold text-ink">{title}</p>
        {description ? (
          <p className="mx-auto mt-1 max-w-sm text-xs text-ink-muted">{description}</p>
        ) : null}
      </div>
      {action}
    </div>
  )
}

export function ErrorState({
  title = 'Something went wrong',
  description,
  onRetry,
  offline = false,
}: {
  title?: string
  description?: string
  onRetry?: () => void
  offline?: boolean
}) {
  return (
    <div role="alert" className="flex flex-col items-center justify-center gap-3 px-6 py-14 text-center">
      <div className="grid size-12 place-items-center rounded-full bg-emergency/10 text-emergency">
        {offline ? <WifiOff className="size-5" aria-hidden /> : <AlertTriangle className="size-5" aria-hidden />}
      </div>
      <div>
        <p className="text-sm font-semibold text-ink">{title}</p>
        {description ? (
          <p className="mx-auto mt-1 max-w-sm text-xs text-ink-muted">{description}</p>
        ) : null}
      </div>
      {onRetry ? (
        <Button size="sm" variant="secondary" onClick={onRetry}>
          Try again
        </Button>
      ) : null}
    </div>
  )
}

/** Offline / connectivity banner — shown when a request fails without a status. */
export function OfflineBanner() {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn">
      <WifiOff className="size-4" aria-hidden />
      <span>You appear to be offline. Showing the last data we have; changes will need a connection.</span>
    </div>
  )
}
