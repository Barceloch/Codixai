// The light and dark theme of the audit interface.
//
// The preference is system, light or dark; it is persisted in localStorage
// and the system preference is the fallback. The daisyUI theme is selected
// with the data-theme attribute of the document, which the CSS of daisyUI
// reads. The inline script of index.html applies the same decision before
// the first paint, so the window never flashes the wrong theme.

export type ThemePreference = 'system' | 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'codixai.theme'

// readPreference returns the persisted preference, or "system".
function readPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      return stored
    }
  } catch {
    // Storage can be unavailable; the system preference is enough.
  }
  return 'system'
}

class Theme {
  preference = $state<ThemePreference>(readPreference())
  resolved = $state<ResolvedTheme>('light')

  private media: MediaQueryList | null = null

  constructor() {
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      this.media = window.matchMedia('(prefers-color-scheme: dark)')
      this.media.addEventListener('change', () => {
        if (this.preference === 'system') {
          this.apply()
        }
      })
    }
    this.apply()
  }

  // setPreference changes the theme, persists it and applies it.
  setPreference(preference: ThemePreference): void {
    this.preference = preference
    try {
      localStorage.setItem(STORAGE_KEY, preference)
    } catch {
      // The theme still changes for this session.
    }
    this.apply()
  }

  // apply writes the resolved theme on the document.
  apply(): void {
    const systemDark = this.media ? this.media.matches : false
    this.resolved =
      this.preference === 'system'
        ? systemDark
          ? 'dark'
          : 'light'
        : this.preference
    if (typeof document !== 'undefined') {
      document.documentElement.dataset.theme = this.resolved
    }
  }
}

export const theme = new Theme()
