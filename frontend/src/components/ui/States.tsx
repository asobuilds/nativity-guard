import type { ReactNode } from 'react'
import { AlertTriangle, Inbox, Loader2, WifiOff } from 'lucide-react'
import { cn } from '@/lib/cn'
import { extractErrorMessage, isOfflineError } from '@/lib/queryClient'
import { Button } from './Button'

export function Spinner({ className, label }: { className?: string; label?: string }) {
  if (label) {
    return (
      <span role="status" aria-live="polite" className="inline-flex items-center gap-2">
        <Loader2 className={cn('size-5 animate-spin text-ink-muted', className)} aria-hidden />
        <span className="sr-only">{label}</span>
      </span>
    )
  }
  return <Loader2 className={cn('size-5 animate-spin text-ink-muted', className)} aria-hidden />
}

export function FullPageSpinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
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

/** Accessible loading state for lists, panels, and async sections. */
export function LoadingState({
  label = 'Loading…',
  count = 3,
  variant = 'skeleton',
  className,
  itemClassName,
}: {
  label?: string
  count?: number
  variant?: 'skeleton' | 'spinner'
  className?: string
  itemClassName?: string
}) {
  if (variant === 'spinner') {
    return (
      <div
        role="status"
        aria-live="polite"
        aria-busy="true"
        className={cn('flex flex-col items-center justify-center gap-3 px-6 py-14 text-center text-ink-muted', className)}
      >
        <Loader2 className="size-5 animate-spin" aria-hidden />
        <span className="text-sm">{label}</span>
      </div>
    )
  }

  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className={cn('flex flex-col gap-3', className)}
    >
      <span className="sr-only">{label}</span>
      {Array.from({ length: Math.max(1, count) }, (_, index) => (
        <Skeleton key={index} className={cn('h-16 w-full', itemClassName)} />
      ))}
    </div>
  )
}

export function EmptyState({
  title,
  description,
  icon,
  action,
  className,
}: {
  title: string
  description?: string
  icon?: ReactNode
  action?: ReactNode
  className?: string
}) {
  return (
    <div
      role="status"
      className={cn('flex flex-col items-center justify-center gap-3 px-6 py-14 text-center', className)}
    >
      <div aria-hidden className="grid size-12 place-items-center rounded-full bg-surface-hi text-ink-faint">
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
  title,
  description,
  error,
  onRetry,
  retrying = false,
  retryLabel = 'Try again',
  offline,
  action,
  className,
}: {
  title?: string
  description?: string
  error?: unknown
  onRetry?: () => void
  retrying?: boolean
  retryLabel?: string
  offline?: boolean
  action?: ReactNode
  className?: string
}) {
  const resolvedOffline = offline ?? (error !== undefined ? isOfflineError(error) : false)
  const resolvedTitle = title ?? (resolvedOffline ? 'You appear to be offline' : 'Something went wrong')
  const resolvedDescription = description ?? (error !== undefined ? extractErrorMessage(error) : undefined)

  return (
    <div
      role="alert"
      className={cn('flex flex-col items-center justify-center gap-3 px-6 py-14 text-center', className)}
    >
      <div aria-hidden className="grid size-12 place-items-center rounded-full bg-emergency/10 text-emergency">
        {resolvedOffline ? <WifiOff className="size-5" aria-hidden /> : <AlertTriangle className="size-5" aria-hidden />}
      </div>
      <div>
        <p className="text-sm font-semibold text-ink">{resolvedTitle}</p>
        {resolvedDescription ? (
          <p className="mx-auto mt-1 max-w-sm text-xs text-ink-muted">{resolvedDescription}</p>
        ) : null}
      </div>
      {onRetry || action ? (
        <div className="flex flex-wrap items-center justify-center gap-2">
          {onRetry ? (
            <Button size="sm" variant="secondary" loading={retrying} onClick={onRetry}>
              {retryLabel}
            </Button>
          ) : null}
          {action}
        </div>
      ) : null}
    </div>
  )
}

/** Offline / connectivity banner — shown when a request fails without a status. */
export function OfflineBanner({
  message = 'You appear to be offline. Showing the last data we have; changes will need a connection.',
  className,
}: {
  message?: string
  className?: string
}) {
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        'flex items-center gap-2 rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn',
        className,
      )}
    >
      <WifiOff className="size-4 shrink-0" aria-hidden />
      <span>{message}</span>
    </div>
  )
}
