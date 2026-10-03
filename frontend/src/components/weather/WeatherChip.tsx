import { useState } from 'react'
import {
  Cloud,
  CloudDrizzle,
  CloudFog,
  CloudLightning,
  CloudRain,
  CloudSnow,
  CloudSun,
  Sun,
  Thermometer,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { WeatherPanel } from './WeatherPanel'
import { useWeather } from '@/hooks/useWeather'

export function weatherIcon(name: string | undefined) {
  switch (name) {
    case 'sun':
      return Sun
    case 'partly-cloudy':
      return CloudSun
    case 'cloud':
      return Cloud
    case 'fog':
      return CloudFog
    case 'drizzle':
      return CloudDrizzle
    case 'rain':
      return CloudRain
    case 'snow':
      return CloudSnow
    case 'thunderstorm':
      return CloudLightning
    default:
      return Cloud
  }
}

interface WeatherChipProps {
  latitude: number | null | undefined
  longitude: number | null | undefined
  label: string
  approximate?: boolean
  className?: string
}

/**
 * Compact weather pill. Reads the current conditions and, on click, opens a
 * full-screen panel with the hourly strip, daily forecast, and the LLM
 * advisory. Renders nothing at all if there is no weather to show, so it
 * never occupies empty space on a card.
 */
export function WeatherChip({
  latitude,
  longitude,
  label,
  approximate = false,
  className,
}: WeatherChipProps) {
  const [open, setOpen] = useState(false)
  const query = useWeather(latitude, longitude, label, approximate)

  if (!query.data) return null

  const current = query.data.current
  const Icon = weatherIcon(current.icon)

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={cn(
          'group inline-flex items-center gap-2 rounded-full border border-border bg-surface/70 px-3 py-1.5 text-xs text-ink',
          'backdrop-blur-sm transition-all hover:border-signal/40 hover:bg-signal/5',
          className,
        )}
        aria-label="Open weather details"
      >
        <Icon className="size-4 text-signal" aria-hidden />
        <span className="font-medium tabular-nums">
          {Math.round(current.temperature)}°C
        </span>
        <span className="text-ink-muted">{current.weatherLabel}</span>
        {query.data.approximate ? (
          <span className="rounded-full bg-surface-hi px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-ink-faint">
            approx
          </span>
        ) : null}
      </button>

      <WeatherPanel open={open} onClose={() => setOpen(false)} bundle={query.data} />
    </>
  )
}

export { Thermometer }