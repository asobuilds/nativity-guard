export type Theme = 'forest' | 'midnight' | 'daylight' | 'ocean' | 'stone'
export type TextSize = 'standard' | 'large' | 'xlarge' | 'xxlarge'

export interface Preferences {
  theme: Theme
  textSize: TextSize
  notifyAlerts: boolean
  notifyCaseUpdates: boolean
  notifyCommunityReplies: boolean
}

const KEY = 'nativity-guard-display-preferences-v2'

export const DEFAULT_PREFERENCES: Preferences = {
  theme: 'forest',
  textSize: 'standard',
  notifyAlerts: true,
  notifyCaseUpdates: true,
  notifyCommunityReplies: true,
}

export function isTheme(v: unknown): v is Theme {
  return v === 'forest' || v === 'midnight' || v === 'daylight' || v === 'ocean' || v === 'stone'
}

export function isTextSize(v: unknown): v is TextSize {
  return v === 'standard' || v === 'large' || v === 'xlarge' || v === 'xxlarge'
}

export function normalizePreferences(input: unknown): Preferences {
  const p = (input ?? {}) as Partial<Preferences>
  return {
    theme: isTheme(p.theme) ? p.theme : DEFAULT_PREFERENCES.theme,
    textSize: isTextSize(p.textSize) ? p.textSize : DEFAULT_PREFERENCES.textSize,
    notifyAlerts: typeof p.notifyAlerts === 'boolean' ? p.notifyAlerts : true,
    notifyCaseUpdates: typeof p.notifyCaseUpdates === 'boolean' ? p.notifyCaseUpdates : true,
    notifyCommunityReplies:
      typeof p.notifyCommunityReplies === 'boolean' ? p.notifyCommunityReplies : true,
  }
}

export function readPreferences(): Preferences {
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) ?? '{}') as Partial<Preferences>
    return normalizePreferences(saved)
  } catch {
    return DEFAULT_PREFERENCES
  }
}

export function applyPreferences(preferences: Preferences) {
  document.documentElement.dataset.theme = preferences.theme
  document.documentElement.dataset.textSize = preferences.textSize
}

export function savePreferences(preferences: Preferences) {
  try {
    localStorage.setItem(KEY, JSON.stringify(preferences))
  } catch {
    /* storage unavailable — the current page can still use the preference */
  }
  applyPreferences(preferences)
}