const FADE_MS = 250

const GIVE_UP_MS = 6000

export const hideSplash = () => {
  const splash = document.getElementById('splash')
  if (!splash) return
  splash.style.opacity = '0'
  splash.setAttribute('aria-hidden', 'true')
  window.setTimeout(() => splash.remove(), FADE_MS)
}

export const hideSplashWhenReady = (root) => {
  const deadline = Date.now() + GIVE_UP_MS
  const settle = () => {
    if (root.childElementCount === 0 && Date.now() < deadline) {
      window.requestAnimationFrame(settle)
      return
    }
    hideSplash()
  }
  window.requestAnimationFrame(settle)
}
