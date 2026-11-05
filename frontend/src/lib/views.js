import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { flushViews, trackView } from './analytics'

export const usePageViews = () => {
  const location = useLocation()
  useEffect(() => {
    trackView(location.pathname)
  }, [location.pathname])
  useEffect(() => {
    const leave = () => {
      flushViews().catch(() => null)
    }
    window.addEventListener('pagehide', leave)
    return () => window.removeEventListener('pagehide', leave)
  }, [])
}

export default usePageViews
