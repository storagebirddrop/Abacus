import { useCallback, useEffect, useState } from 'react'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'abacus-theme'

function getInitialTheme(): Theme {
  if (typeof localStorage !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'light' || saved === 'dark') return saved
  }
  // Dark is the app's primary identity — only defer to an explicit OS
  // preference for light mode, not just the absence of a dark preference.
  if (typeof matchMedia !== 'undefined' && matchMedia('(prefers-color-scheme: light)').matches) {
    return 'light'
  }
  return 'dark'
}

function apply(theme: Theme) {
  const root = document.documentElement
  root.classList.toggle('light', theme === 'light')
}

/**
 * useTheme manages the light/dark theme: applies the `light` class to <html>
 * (dark is the unstyled default), persists the choice, and defaults to dark
 * unless the OS explicitly prefers light.
 */
export function useTheme() {
  const [theme, setTheme] = useState<Theme>(getInitialTheme)

  useEffect(() => {
    apply(theme)
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      // ignore storage failures (e.g. private mode)
    }
  }, [theme])

  const toggle = useCallback(() => {
    setTheme((t) => (t === 'dark' ? 'light' : 'dark'))
  }, [])

  return { theme, toggle }
}
