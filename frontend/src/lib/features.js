export const FEATURE_MESSAGES = 'messages'

export const FEATURE_POLLS = 'polls'

export const FEATURE_ATTENDANCE = 'attendance'

export const FEATURE_GROUPS = 'group_chats'

export const FEATURE_WEB_ATTENDANCE = 'web_attendance'

export const FEATURE_LABELS = {
  [FEATURE_MESSAGES]: 'Messages',
  [FEATURE_POLLS]: 'Polls',
  [FEATURE_ATTENDANCE]: 'Attendance',
  [FEATURE_GROUPS]: 'Group chats',
  [FEATURE_WEB_ATTENDANCE]: 'Attendance from the website',
}

export const FEATURE_OFF_CODE = 'feature_off'

const PAGES = [
  { feature: FEATURE_MESSAGES, pattern: /^\/(student|teacher)\/messages(\/|$)/ },
  { feature: FEATURE_POLLS, pattern: /^\/(student|cr)\/polls(\/|$)/ },
  {
    feature: FEATURE_ATTENDANCE,
    pattern: /^\/(student|teacher)\/(attendance|my-attendance|phone-requests)(\/|$)/,
  },
]

const API = [
  { feature: FEATURE_MESSAGES, pattern: /^\/api\/messages(\/|$)/ },
  { feature: FEATURE_POLLS, pattern: /^\/api\/(students|cr)\/polls(\/|$)/ },
  { feature: FEATURE_ATTENDANCE, pattern: /^\/api\/(students|teacher)\/attendance(\/|$)/ },
]

const pathOf = (target) => {
  const value = typeof target === 'function' ? target() : target
  return String(value || '').split('#')[0].split('?')[0]
}

export const featureOn = (features, feature) => {
  if (!feature || !features || typeof features !== 'object') return true
  return features[feature] !== false
}

export const featureOfPage = (target) => {
  const path = pathOf(target)
  return PAGES.find((entry) => entry.pattern.test(path))?.feature || ''
}

export const featureOfRequest = (target) => {
  const path = pathOf(target)
  return API.find((entry) => entry.pattern.test(path))?.feature || ''
}

export const pageOn = (features, target) => featureOn(features, featureOfPage(target))

export const requestOn = (features, target) => featureOn(features, featureOfRequest(target))

export const featureOffMessage = (feature) =>
  `${FEATURE_LABELS[feature] || 'This part of the portal'} is turned off by the department`

export const visibleItems = (features, items) =>
  (items || []).filter((item) => pageOn(features, item.to))

export const visibleGroups = (features, groups) =>
  (groups || [])
    .map((group) => ({ ...group, items: visibleItems(features, group.items) }))
    .filter((group) => group.items.length > 0)
