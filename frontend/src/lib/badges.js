import { useCallback, useEffect, useState } from 'react'
import api, { hasSession } from './api'

export const BADGE_NOTICES = 'notices'

export const BADGE_MATERIALS = 'materials'

export const BADGE_POLLS = 'polls'

export const BADGE_EXAMS = 'exams'

export const BADGE_ROUTINE = 'routine'

export const BADGE_COURSES = 'courses'

export const BADGE_SECTIONS = [
  BADGE_NOTICES,
  BADGE_MATERIALS,
  BADGE_POLLS,
  BADGE_EXAMS,
  BADGE_ROUTINE,
  BADGE_COURSES,
]

const BADGE_EVENT = 'sustcse:badges'

const empty = () => Object.fromEntries(BADGE_SECTIONS.map((section) => [section, 0]))

export const refreshBadges = () => window.dispatchEvent(new Event(BADGE_EVENT))

export const markSectionSeen = async (section) => {
  if (!BADGE_SECTIONS.includes(section) || !hasSession()) return
  await api.post(`/api/me/badges/${section}/seen`).catch(() => null)
  refreshBadges()
}

export const setAppBadge = (count) => {
  if (typeof navigator === 'undefined') return
  try {
    if (count > 0) navigator.setAppBadge?.(count)
    else navigator.clearAppBadge?.()
  } catch {
    return
  }
}

export const useSectionBadges = (enabled = true) => {
  const [counts, setCounts] = useState(empty)
  const load = useCallback(() => {
    if (!enabled || !hasSession()) return undefined
    return api
      .get('/api/me/badges')
      .then((data) => setCounts({ ...empty(), ...(data?.sections || {}) }))
      .catch(() => null)
  }, [enabled])
  useEffect(() => {
    load()
    window.addEventListener(BADGE_EVENT, load)
    document.addEventListener('visibilitychange', load)
    return () => {
      window.removeEventListener(BADGE_EVENT, load)
      document.removeEventListener('visibilitychange', load)
    }
  }, [load])
  return counts
}

export const useSectionSeen = (section) => {
  useEffect(() => {
    if (!section) return
    markSectionSeen(section)
  }, [section])
}
