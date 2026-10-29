import { useEffect } from 'react'

const FIELDS = 'input, textarea, select, [contenteditable="true"]'

const SETTLE = 180

const MARGIN = 16

export const useKeyboardAwareFocus = () => {
  useEffect(() => {
    const viewport = window.visualViewport
    if (!viewport) return undefined
    let timer = 0
    const reveal = () => {
      const active = document.activeElement
      if (!active || typeof active.matches !== 'function' || !active.matches(FIELDS)) return
      const box = active.getBoundingClientRect()
      const top = viewport.offsetTop + MARGIN
      const bottom = viewport.offsetTop + viewport.height - MARGIN
      if (box.top >= top && box.bottom <= bottom) return
      active.scrollIntoView({ block: 'center', behavior: 'smooth' })
    }
    const schedule = () => {
      window.clearTimeout(timer)
      timer = window.setTimeout(reveal, SETTLE)
    }
    document.addEventListener('focusin', schedule)
    viewport.addEventListener('resize', schedule)
    return () => {
      window.clearTimeout(timer)
      document.removeEventListener('focusin', schedule)
      viewport.removeEventListener('resize', schedule)
    }
  }, [])
}

export default useKeyboardAwareFocus
