import { useEffect, useState } from 'react'

export const isStandalone = () => {
  try {
    return (
      window.matchMedia('(display-mode: standalone)').matches ||
      window.matchMedia('(display-mode: fullscreen)').matches ||
      window.navigator.standalone === true
    )
  } catch {
    return false
  }
}

export const useStandalone = () => {
  const [installed, setInstalled] = useState(isStandalone)
  useEffect(() => {
    let media = null
    try {
      media = window.matchMedia('(display-mode: standalone)')
    } catch {
      return undefined
    }
    const onChange = () => setInstalled(isStandalone())
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])
  return installed
}
