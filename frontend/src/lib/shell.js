import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { isNative } from './native'
import { CHROME_COLORS, THEME_DARK } from './theme'

const HOME_PATHS = ['/', '/student/today', '/teacher/upcoming-routine', '/student/login']

export const applyNativeChrome = async (resolved) => {
  if (!isNative()) return
  const { StatusBar, Style } = await import('@capacitor/status-bar')
  const bar = CHROME_COLORS[resolved] || CHROME_COLORS.light
  await StatusBar.setOverlaysWebView({ overlay: false }).catch(() => null)
  await StatusBar.setBackgroundColor({ color: bar }).catch(() => null)
  await StatusBar.setStyle({
    style: resolved === THEME_DARK ? Style.Dark : Style.Light,
  }).catch(() => null)
}

export const useNativeShell = (resolved) => {
  useEffect(() => {
    if (!isNative()) return undefined
    let live = true
    let listener = null
    applyNativeChrome(resolved).catch(() => null)
    import('@capacitor/app').then(({ App }) => {
      if (!live) return
      App.addListener('appStateChange', ({ isActive }) => {
        if (isActive) applyNativeChrome(resolved).catch(() => null)
      }).then((handle) => {
        if (live) listener = handle
        else handle.remove()
      })
    })
    return () => {
      live = false
      if (listener) listener.remove()
    }
  }, [resolved])
}

export const useNativeBackButton = () => {
  const navigate = useNavigate()
  const location = useLocation()
  useEffect(() => {
    if (!isNative()) return undefined
    let listener = null
    let live = true
    import('@capacitor/app').then(({ App }) => {
      if (!live) return
      App.addListener('backButton', ({ canGoBack }) => {
        if (HOME_PATHS.includes(window.location.pathname) || !canGoBack) {
          App.exitApp()
          return
        }
        navigate(-1)
      }).then((handle) => {
        if (live) listener = handle
        else handle.remove()
      })
    })
    return () => {
      live = false
      if (listener) listener.remove()
    }
  }, [navigate, location.pathname])
}
