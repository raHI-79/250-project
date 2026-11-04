const NO_BODY = { style: {} }

const pageBody = () => (typeof document === 'undefined' ? NO_BODY : document.body || NO_BODY)

export const createScrollLock = (bodyOf = pageBody) => {
  let depth = 0
  let saved = ''
  const apply = () => {
    const { style } = bodyOf()
    saved = style.overflow || ''
    style.overflow = 'hidden'
  }
  const restore = () => {
    bodyOf().style.overflow = saved
    saved = ''
  }
  const lock = () => {
    if (depth === 0) apply()
    depth += 1
    let freed = false
    return () => {
      if (freed) return
      freed = true
      depth -= 1
      if (depth > 0) return
      depth = 0
      restore()
    }
  }
  return {
    lock,
    locked: () => depth > 0,
    depth: () => depth,
    reset: () => {
      if (depth > 0) restore()
      depth = 0
    },
  }
}

const shared = createScrollLock()

export const lockScroll = () => shared.lock()

export const isScrollLocked = () => shared.locked()

export const scrollLockDepth = () => shared.depth()

export const resetScrollLock = () => shared.reset()

export default shared
