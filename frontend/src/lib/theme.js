import { readRaw, writeRaw } from './storage'

export const THEME_LIGHT = 'light'

export const THEME_DARK = 'dark'

export const THEME_SYSTEM = 'system'

export const THEME_KEY = 'sustcse.appearance'

export const THEME_CHOICES = [
  { value: THEME_LIGHT, label: 'Light' },
  { value: THEME_DARK, label: 'Dark' },
  { value: THEME_SYSTEM, label: 'System' },
]

export const THEME_COLORS = { light: '#F7F8FA', dark: '#0E1014' }

export const CHROME_COLORS = { light: '#F7F8FA', dark: '#0E1014' }

const DARK_QUERY = '(prefers-color-scheme: dark)'

export const darkQuery = () => {
  try {
    return window.matchMedia(DARK_QUERY)
  } catch {
    return null
  }
}

export const prefersDark = () => Boolean(darkQuery()?.matches)

export const storedChoice = () => {
  const raw = readRaw(THEME_KEY)
  if (raw === THEME_LIGHT || raw === THEME_DARK || raw === THEME_SYSTEM) return raw
  return THEME_SYSTEM
}

export const storeChoice = (choice) => writeRaw(THEME_KEY, choice)

export const resolveTheme = (choice, systemDark) => {
  if (choice === THEME_LIGHT || choice === THEME_DARK) return choice
  return systemDark ? THEME_DARK : THEME_LIGHT
}

export const applyTheme = (resolved) => {
  const root = document.documentElement
  root.dataset.theme = resolved
  root.style.colorScheme = resolved
  document
    .querySelectorAll('meta[name="theme-color"]')
    .forEach((tag) => tag.setAttribute('content', THEME_COLORS[resolved] || THEME_COLORS.light))
}

export const bootstrapTheme = () => applyTheme(resolveTheme(storedChoice(), prefersDark()))
