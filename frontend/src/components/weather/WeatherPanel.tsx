import { useEffect } from 'react'
import { AlertTriangle, Droplets, MapPin, Wind, X } from 'lucide-react'
import type { WeatherBundle } from '@/hooks/useWeather'
import { weatherIcon } from './WeatherChip'

interface WeatherPanelProps {
  open: boolean
  onClose: () => void
  bundle: WeatherBundle
}

/**
 * Full weather view. Modal on all screen sizes — the same content is used
 * inline in the map overlay when the caller wants a persistent view, but the
 * chip always opens this modal.
 */
export function WeatherPanel({ open, onClose, bundle }: WeatherPanelProps) {
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const { current, hourly, daily, advisory, alert, location, approximate } = bundle
  const CurrentIcon = weatherIcon(current.icon)

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Weather"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex max-h-[88vh] w-full max-w-lg flex-col overflow-hidden rounded-3xl border border-border bg-base shadow-2xl">
        <header className="relative border-b border-border bg-gradient-to-br from-signal/10 to-transparent p-5">
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="absolute right-4 top-4 grid size-8 place-items-center rounded-full text-ink-muted hover:bg-surface-hi"
          >
            <X className="size-4" aria-hidden />
          </button>

          <p className="flex items-center gap-1.5 text-xs text-ink-muted">
            <MapPin className="size-3.5" aria-hidden />
            {location}
            {approximate ? <span className="text-ink-faint">· approximate</span> : null}
          </p>

          <div className="mt-4 flex items-center gap-5">
            <CurrentIcon className="size-14 shrink-0 text-signal" aria-hidden />
            <div>
              <p className="text-4xl font-bold tabular-nums text-ink">
                {Math.round(current.temperature)}°
              </p>
              <p className="text-sm text-ink-muted">{current.weatherLabel}</p>
              <p className="mt-0.5 text-xs text-ink-faint">
                Feels like {Math.round(current.apparentTemperature)}°
              </p>
            </div>
          </div>

          <div className="mt-4 flex flex-wrap gap-4 text-xs text-ink-muted">
            <span className="flex items-center gap-1.5">
              <Wind className="size-3.5" aria-hidden />
              {Math.round(current.windKph)} km/h
            </span>
            <span className="flex items-center gap-1.5">
              <Droplets className="size-3.5" aria-hidden />
              {Math.round(current.humidity)}%
            </span>
          </div>
        </header>

        <div className="overflow-y-auto">
          {alert ? (
            <div className="border-b border-warn/30 bg-warn/10 px-5 py-3">
              <p className="flex items-start gap-2 text-sm text-warn">
                <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
                <span>{alert}</span>
              </p>
            </div>
          ) : null}

          {advisory ? (
            <section className="border-b border-border px-5 py-4">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-signal">
                What this means
              </p>
              <p className="mt-1.5 whitespace-pre-wrap text-sm leading-relaxed text-ink-muted">
                {advisory}
              </p>
            </section>
          ) : null}

          {hourly.length > 0 ? (
            <section className="border-b border-border px-5 py-4">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-ink-faint">
                Next 12 hours
              </p>
              <div className="mt-3 flex gap-4 overflow-x-auto pb-2">
                {hourly.map((h) => {
                  const Icon = weatherIcon(h.icon)
                  const hour = new Date(h.time).getHours()
                  const label =
                    hour === 0
                      ? '12a'
                      : hour < 12
                        ? `${hour}a`
                        : hour === 12
                          ? '12p'
                          : `${hour - 12}p`
                  return (
                    <div
                      key={h.time}
                      className="flex min-w-[48px] shrink-0 flex-col items-center gap-1.5 text-center"
                    >
                      <span className="text-[10px] uppercase tracking-wide text-ink-faint">
                        {label}
                      </span>
                      <Icon className="size-5 text-signal" aria-hidden />
                      <span className="text-xs font-medium tabular-nums text-ink">
                        {Math.round(h.temperature)}°
                      </span>
                      {h.precipitationProbability > 20 ? (
                        <span className="text-[10px] text-signal">
                          {Math.round(h.precipitationProbability)}%
                        </span>
                      ) : null}
                    </div>
                  )
                })}
              </div>
            </section>
          ) : null}

          {daily.length > 0 ? (
            <section className="px-5 py-4">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-ink-faint">
                7-day forecast
              </p>
              <ul className="mt-3 space-y-2">
                {daily.map((d) => {
                  const Icon = weatherIcon(d.icon)
                  const day = new Date(d.date).toLocaleDateString(undefined, {
                    weekday: 'short',
                  })
                  return (
                    <li key={d.date} className="flex items-center gap-3">
                      <span className="w-10 text-xs text-ink-muted">{day}</span>
                      <Icon className="size-5 text-signal" aria-hidden />
                      <span className="flex-1 text-xs text-ink-muted">
                        {d.weatherLabel}
                      </span>
                      {d.precipitationProbability > 20 ? (
                        <span className="text-[11px] tabular-nums text-signal">
                          {Math.round(d.precipitationProbability)}%
                        </span>
                      ) : null}
                      <span className="text-xs font-medium tabular-nums text-ink">
                        {Math.round(d.tempMax)}°
                        <span className="text-ink-faint"> / {Math.round(d.tempMin)}°</span>
                      </span>
                    </li>
                  )
                })}
              </ul>
            </section>
          ) : null}
        </div>

        <footer className="border-t border-border bg-surface-hi/30 px-5 py-3 text-[11px] text-ink-faint">
          Data from{' '}
          <a
            href="https://open-meteo.com/"
            target="_blank"
            rel="noopener noreferrer"
            className="text-signal hover:underline"
          >
            Open-Meteo
          </a>
          . Advisory may be imperfect — treat it as guidance, not instruction.
        </footer>
      </div>
    </div>
  )
}