import { useEffect, type KeyboardEvent, type PointerEvent } from 'react'
import { Mic, MicOff, Radio, ShieldCheck, Volume2, VolumeX, X } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { useWalkieTalkie } from '@/hooks/useWalkieTalkie'
import { cn } from '@/lib/cn'
import { formatCountdown, type WalkieChannel } from '@/lib/walkie'
import { Card } from '@/components/ui/Card'

interface WalkieTalkieProps {
  channel: WalkieChannel | null
  channelName?: string
  variant?: 'page' | 'dock'
  onClose?: () => void
}

/**
 * Unit walkie-talkie.
 *
 * Hold to talk: the floor request is sent first, audio only flows after the
 * server grants it, and releasing sends the floor back. A busy unit says who is
 * speaking instead of silently pretending to transmit.
 */
export function WalkieTalkie({ channel, channelName, variant = 'page', onClose }: WalkieTalkieProps) {
  const { role } = useAuth()
  const {
    status,
    speakerName,
    error,
    online,
    secondsLeft,
    muted,
    setMuted,
    override,
    setOverride,
    press,
    release,
    enablePlayback,
  } = useWalkieTalkie({ channel })

  const connected = status !== 'disconnected' && status !== 'connecting' && status !== 'error'
  const transmitting = status === 'transmitting'
  const busy = status === 'busy'
  const isSuperAdmin = role === 'super_admin'

  useEffect(() => {
    const stop = () => release()
    const onVisibility = () => {
      if (document.visibilityState === 'hidden') stop()
    }
    window.addEventListener('blur', stop)
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.removeEventListener('blur', stop)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [release])

  function handlePress(event: PointerEvent<HTMLButtonElement>) {
    event.preventDefault()
    void enablePlayback()
    if (event.currentTarget.setPointerCapture) {
      try {
        event.currentTarget.setPointerCapture(event.pointerId)
      } catch {
        /* pointer may already be gone */
      }
    }
    press()
  }

  function handleKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key !== ' ' && event.key !== 'Enter') return
    event.preventDefault()
    if (event.repeat) return
    void enablePlayback()
    press()
  }

  function handleKeyUp(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key !== ' ' && event.key !== 'Enter') return
    event.preventDefault()
    release()
  }

  const statusText = (() => {
    switch (status) {
      case 'connecting':
        return 'Connecting to the channel…'
      case 'disconnected':
        return 'Channel offline'
      case 'error':
        return error ?? 'Channel error'
      case 'requesting':
        return 'Requesting the channel…'
      case 'transmitting':
        return `You are transmitting · ${formatCountdown(secondsLeft)} left`
      case 'busy':
        return `${speakerName || 'Another officer'} is transmitting`
      default:
        return 'Channel clear — hold to talk'
    }
  })()

  return (
    <Card
      className={cn(
        'overflow-hidden',
        variant === 'page' ? 'mx-auto w-full max-w-3xl' : 'w-full shadow-panel',
      )}
    >
      <header className="flex items-start justify-between gap-3 border-b border-border px-4 py-3">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={cn(
              'grid size-9 shrink-0 place-items-center rounded-full',
              transmitting ? 'bg-emergency/15 text-emergency' : 'bg-signal/10 text-signal',
            )}
          >
            <Radio className="size-5" aria-hidden />
          </span>
          <div className="min-w-0">
            <h2 className="truncate text-sm font-semibold text-ink">Unit walkie-talkie</h2>
            <p className="truncate text-xs text-ink-muted">
              {channelName || 'Channel'} ·{' '}
              <span className={cn(connected ? 'text-signal' : 'text-ink-faint')}>
                {connected ? `${online} online` : 'offline'}
              </span>
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button
            type="button"
            onClick={() => setMuted(!muted)}
            aria-label={muted ? 'Unmute incoming audio' : 'Mute incoming audio'}
            aria-pressed={muted}
            className="rounded-md p-2 text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink"
          >
            {muted ? <VolumeX className="size-4" aria-hidden /> : <Volume2 className="size-4" aria-hidden />}
          </button>
          {onClose ? (
            <button
              type="button"
              onClick={onClose}
              aria-label="Close walkie-talkie"
              className="rounded-md p-2 text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink"
            >
              <X className="size-4" aria-hidden />
            </button>
          ) : null}
        </div>
      </header>

      <div className={cn('flex flex-col items-center gap-4', variant === 'page' ? 'p-6' : 'p-4')}>
        <p
          role="status"
          aria-live="polite"
          className={cn(
            'min-h-5 text-center text-sm',
            status === 'error' ? 'text-emergency' : transmitting ? 'text-emergency' : 'text-ink-muted',
          )}
        >
          {statusText}
        </p>

        <button
          type="button"
          onPointerDown={handlePress}
          onPointerUp={release}
          onPointerLeave={release}
          onPointerCancel={release}
          onLostPointerCapture={release}
          onKeyDown={handleKeyDown}
          onKeyUp={handleKeyUp}
          onContextMenu={(event) => event.preventDefault()}
          disabled={!connected}
          aria-pressed={transmitting}
          aria-label="Hold to talk"
          className={cn(
            'relative grid touch-none select-none place-items-center rounded-full border-2 transition-all',
            'focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-signal',
            'disabled:cursor-not-allowed disabled:opacity-50',
            variant === 'page' ? 'size-40' : 'size-28',
            transmitting
              ? 'border-emergency bg-emergency text-white shadow-[0_0_0_12px_rgba(220,38,38,0.12)]'
              : busy
                ? 'border-border-hi bg-surface-hi text-ink-muted'
                : 'border-signal/40 bg-signal/10 text-signal hover:bg-signal/15 active:scale-[0.98]',
          )}
        >
          {busy ? <MicOff className="size-10" aria-hidden /> : <Mic className="size-10" aria-hidden />}
          <span className="mt-1 text-xs font-semibold uppercase tracking-wide">
            {transmitting ? 'Live' : busy ? 'Busy' : 'Hold to talk'}
          </span>
        </button>

        {transmitting ? (
          <div
            className="h-1.5 w-full max-w-xs overflow-hidden rounded-full bg-surface-hi"
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={60}
            aria-valuenow={secondsLeft}
            aria-label="Transmission time remaining"
          >
            <div
              className="h-full rounded-full bg-emergency transition-[width] duration-200"
              style={{ width: `${Math.min(100, Math.max(0, (secondsLeft / 60) * 100))}%` }}
            />
          </div>
        ) : null}

        {isSuperAdmin ? (
          <label className="flex w-full max-w-sm items-center justify-between gap-3 rounded-lg border border-border bg-surface-hi/50 px-3 py-2 text-xs">
            <span className="flex items-center gap-2 text-ink-muted">
              <ShieldCheck className="size-4 text-signal" aria-hidden />
              Command override {override ? 'enabled' : 'disabled'}
            </span>
            <input
              type="checkbox"
              checked={override}
              onChange={(event) => setOverride(event.target.checked)}
              className="size-4 accent-[var(--color-signal)]"
            />
          </label>
        ) : null}

        <p className="text-center text-[11px] text-ink-faint">
          Live audio is not recorded. The channel releases automatically after 60 seconds.
        </p>
      </div>
    </Card>
  )
}
