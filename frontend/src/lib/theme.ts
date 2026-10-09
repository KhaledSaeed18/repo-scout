export type ThemePreference = 'system' | 'light' | 'dark'

export const themeStorageKey = 'repo-scout-theme'

const darkQuery = '(prefers-color-scheme: dark)'

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'system' || value === 'light' || value === 'dark'
}

/** Applies the preference to <html> and remembers it for the next page load. */
export function applyTheme(preference: ThemePreference) {
  const dark =
    preference === 'dark' || (preference === 'system' && window.matchMedia(darkQuery).matches)
  document.documentElement.classList.toggle('dark', dark)
  localStorage.setItem(themeStorageKey, preference)
}

/** Re-applies a system preference whenever the OS appearance changes. */
export function watchSystemTheme(onChange: () => void): () => void {
  const media = window.matchMedia(darkQuery)
  media.addEventListener('change', onChange)
  return () => media.removeEventListener('change', onChange)
}
