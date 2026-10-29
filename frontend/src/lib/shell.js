import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { isNative } from './native'

const HOME_PATHS = ['/', '/student/today', '/teacher/upcoming-routine', '/student/login']

export const useNativeShell = () => {
  useEffect(() => {
    if (!isNative()) return undefined
    let cancelled = false
    import('@capacitor/status-bar').then(({ StatusBar, Style }) => {
      if (cancelled) return
      StatusBar.setOverlaysWebView({ overlay: false }).catch(() => null)
      StatusBar.setBackgroundColor({ color: '#7A1F2B' }).catch(() => null)
      StatusBar.setStyle({ style: Style.Dark }).catch(() => null)
    })
    return () => {
      cancelled = true
    }
  }, [])
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
