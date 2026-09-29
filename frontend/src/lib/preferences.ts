export type Accent = 'dawn' | 'sky' | 'forest'
export type TextSize = 'standard' | 'large'
export type Preferences = { accent: Accent; textSize: TextSize }

const KEY = 'nativity-guard-display-preferences-v1'
export const DEFAULT_PREFERENCES: Preferences = { accent: 'dawn', textSize: 'standard' }

export function readPreferences(): Preferences {
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) ?? '{}') as Partial<Preferences>
    return {
      accent: saved.accent === 'sky' || saved.accent === 'forest' ? saved.accent : 'dawn',
      textSize: saved.textSize === 'large' ? 'large' : 'standard',
    }
  } catch {
    return DEFAULT_PREFERENCES
  }
}

export function applyPreferences(preferences: Preferences) {
  document.documentElement.dataset.accent = preferences.accent
  document.documentElement.dataset.textSize = preferences.textSize
}

export function savePreferences(preferences: Preferences) {
  try {
    localStorage.setItem(KEY, JSON.stringify(preferences))
  } catch {
    // Storage may be unavailable; the current page can still use the preference.
  }
  applyPreferences(preferences)
}
