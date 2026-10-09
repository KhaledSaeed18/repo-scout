import { useEffect } from 'react'
import { useSettings } from '@/lib/api'
import { applyTheme, isThemePreference, watchSystemTheme } from '@/lib/theme'

/** Keeps the document theme in step with the saved setting and the OS. */
export default function ThemeSync() {
  const { data } = useSettings()
  const preference = isThemePreference(data?.theme) ? data.theme : undefined

  useEffect(() => {
    if (!preference) return
    applyTheme(preference)
    if (preference !== 'system') return
    return watchSystemTheme(() => applyTheme('system'))
  }, [preference])

  return null
}
