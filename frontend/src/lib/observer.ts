import type { ObserverLocation } from '@/types/api'

const CATEGORY_LABEL: Record<string, string> = {
  infrastructure: 'Infrastructure',
  vandalism: 'Vandalism',
  disaster: 'Disaster',
  fire: 'Fire',
  road_accident: 'Road accident',
  traffic: 'Traffic',
  uncategorized: 'Uncategorised',
}

export type ObserverBadgeTone = 'neutral' | 'signal' | 'ok' | 'warn' | 'emergency'

export function observerCategoryLabel(type: string): string {
  const key = type.trim().toLowerCase()
  if (!key) return CATEGORY_LABEL.uncategorized
  return CATEGORY_LABEL[key] ?? type
}

export function observerStatusLabel(status: string): string {
  const trimmed = status.trim()
  if (!trimmed) return 'Unknown'
  return trimmed
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (letter) => letter.toUpperCase())
}

export function observerStatusTone(status: string): ObserverBadgeTone {
  switch (status.trim().toLowerCase()) {
    case 'resolved':
    case 'closed':
      return 'ok'
    case 'escalated':
      return 'emergency'
    case 'assigned':
    case 'dispatched':
    case 'on_scene':
      return 'warn'
    case 'pending':
    case 'investigating':
      return 'signal'
    default:
      return 'neutral'
  }
}

export function observerLocationLabel(location: ObserverLocation): string {
  const parts = [location.ward, location.lga, location.state]
    .map((part) => part?.trim())
    .filter((part): part is string => Boolean(part))
  return parts.join(', ') || 'Location withheld'
}
