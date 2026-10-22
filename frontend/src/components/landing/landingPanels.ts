import type { LandingPanel } from './LandingNavbar'

export const HASH_TO_PANEL: Record<string, LandingPanel> = {
  map: 'map',
  'case-lifecycle': 'track',
  track: 'track',
  units: 'units',
  'sos-simulator': 'sos-simulator',
  'community-alerts': 'community-alerts',
  'ai-safety': 'ai-safety',
  governance: 'governance',
  transparency: 'transparency',
}

export const PANEL_TO_HASH: Record<LandingPanel, string> = {
  map: 'map',
  track: 'case-lifecycle',
  units: 'units',
  'sos-simulator': 'sos-simulator',
  'community-alerts': 'community-alerts',
  'ai-safety': 'ai-safety',
  governance: 'governance',
  transparency: 'transparency',
}

export const PANEL_TITLES: Record<LandingPanel, string> = {
  map: 'Safety Map',
  track: 'Track a Case',
  units: 'Security Units',
  'sos-simulator': 'Emergency SOS Guidance',
  'community-alerts': 'Community Alerts',
  'ai-safety': 'AI Safety Advisor',
  governance: 'Governance',
  transparency: 'Financial Transparency',
}

/** Old landing hashes remain valid, including incoming /#case-lifecycle links.
 * Malformed URLs must not crash the whole public landing page.
 */
export function panelFromHash(hash: string): LandingPanel | null {
  try {
    return HASH_TO_PANEL[decodeURIComponent(hash.replace(/^#/, ''))] ?? null
  } catch {
    return null
  }
}
