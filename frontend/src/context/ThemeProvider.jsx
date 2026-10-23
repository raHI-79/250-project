import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  applyTheme,
  darkQuery,
  prefersDark,
  resolveTheme,
  storeChoice,
  storedChoice,
} from '../lib/theme'
import { ThemeContext } from './theme'

export default function ThemeProvider({ children }) {
  const [choice, setChoice] = useState(storedChoice)
  const [systemDark, setSystemDark] = useState(prefersDark)
  useEffect(() => {
    const media = darkQuery()
    if (!media) return undefined
    const onChange = (event) => setSystemDark(event.matches)
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])
  const resolved = resolveTheme(choice, systemDark)
  useEffect(() => {
    applyTheme(resolved)
  }, [resolved])
  const select = useCallback((next) => {
    storeChoice(next)
    setChoice(next)
  }, [])
  const value = useMemo(() => ({ choice, resolved, select }), [choice, resolved, select])
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}
