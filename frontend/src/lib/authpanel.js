export const APP_AUTH_TITLE = 'SUST CSE Portal'

export const APP_AUTH_SUBTITLE = 'Classes, exams and attendance in one place'

export const APP_AUTH_FOOTER = 'Department of CSE, SUST'

export const appPanel = (title, subtitle) => {
  const own = String(title || '').trim()
  if (!own) return { title: APP_AUTH_TITLE, subtitle: APP_AUTH_SUBTITLE }
  return { title: own, subtitle: String(subtitle || '').trim() }
}

export const appFooter = (version) =>
  `${APP_AUTH_FOOTER} · Version ${String(version || '').trim() || '0.0.0'}`

export default appPanel
