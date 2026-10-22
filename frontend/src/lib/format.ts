/** Formatting helpers. All user-facing dates go through here. */

const dateFmt = new Intl.DateTimeFormat(undefined, {
  day: '2-digit',
  month: 'short',
  year: 'numeric',
})

const dateTimeFmt = new Intl.DateTimeFormat(undefined, {
  day: '2-digit',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

const timeFmt = new Intl.DateTimeFormat(undefined, {
  hour: '2-digit',
  minute: '2-digit',
})

export function toDate(value: string | Date | undefined | null): Date | null {
  if (!value) return null
  const d = value instanceof Date ? value : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDate(value: string | Date | undefined | null): string {
  const d = toDate(value)
  return d ? dateFmt.format(d) : '—'
}

export function formatDateTime(value: string | Date | undefined | null): string {
  const d = toDate(value)
  return d ? dateTimeFmt.format(d) : '—'
}

export function formatTime(value: string | Date | undefined | null): string {
  const d = toDate(value)
  return d ? timeFmt.format(d) : '—'
}

const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

/** "3 hours ago", "in 2 days". Falls back to a date for anything old. */
export function relativeTime(value: string | Date | undefined | null): string {
  const d = toDate(value)
  if (!d) return '—'
  const diffMs = d.getTime() - Date.now()
  const abs = Math.abs(diffMs)

  const minute = 60_000
  const hour = 60 * minute
  const day = 24 * hour

  if (abs < minute) return 'just now'
  if (abs < hour) return rtf.format(Math.round(diffMs / minute), 'minute')
  if (abs < day) return rtf.format(Math.round(diffMs / hour), 'hour')
  if (abs < 7 * day) return rtf.format(Math.round(diffMs / day), 'day')
  return formatDate(d)
}

/** Elapsed duration between two timestamps, e.g. "2h 14m" — used for response metrics. */
export function durationBetween(
  from: string | Date | undefined | null,
  to: string | Date | undefined | null,
): string {
  const a = toDate(from)
  const b = toDate(to)
  if (!a || !b) return '—'
  const ms = Math.abs(b.getTime() - a.getTime())
  const mins = Math.floor(ms / 60_000)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  const rem = mins % 60
  if (hours < 24) return `${hours}h ${rem}m`
  const days = Math.floor(hours / 24)
  return `${days}d ${hours % 24}h`
}

/** Road distance, e.g. "850 m" or "12.4 km". */
export function formatDistance(meters: number | undefined | null): string {
  if (meters == null || !Number.isFinite(meters) || meters < 0) return '—'
  if (meters < 1000) return `${Math.round(meters)} m`
  return `${(meters / 1000).toFixed(1)} km`
}

/** Travel time, e.g. "8 min", "1 h 12 min". */
export function formatDuration(seconds: number | undefined | null): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) return '—'
  const mins = Math.floor(seconds / 60)
  if (mins < 1) return 'less than a minute'
  if (mins < 60) return `${mins} min`
  const hours = Math.floor(mins / 60)
  const rem = mins % 60
  return rem === 0 ? `${hours} h` : `${hours} h ${rem} min`
}

export function initials(first?: string, last?: string): string {
  const a = first?.trim()?.[0] ?? ''
  const b = last?.trim()?.[0] ?? ''
  return (a + b).toUpperCase() || '?'
}

export function fullName(first?: string, last?: string): string {
  const name = [first, last].filter(Boolean).join(' ').trim()
  return name || 'Unknown'
}

/** Compact coordinate display: "6.6111° N, 3.5074° E". */
export function formatCoord(lat?: number, lng?: number): string {
  if (lat === undefined || lng === undefined || (lat === 0 && lng === 0)) return '—'
  const ns = lat >= 0 ? 'N' : 'S'
  const ew = lng >= 0 ? 'E' : 'W'
  return `${Math.abs(lat).toFixed(4)}° ${ns}, ${Math.abs(lng).toFixed(4)}° ${ew}`
}

export function truncate(text: string, max = 120): string {
  return text.length > max ? `${text.slice(0, max - 1).trimEnd()}…` : text
}
