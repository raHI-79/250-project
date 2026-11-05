export const HUMAN_FORMS = [
  '/student/login',
  '/student/register',
  '/student/forgot-password',
  '/teacher/login',
  '/teacher/forgot-password',
  '/teacher/activate',
  '/office-portal',
  '/office-portal/forgot-password',
  '/office-portal/accept-invite',
  '/control-room',
  '/control-room/forgot-password',
]

export const needsHumanCheck = (path) => HUMAN_FORMS.includes(String(path || ''))

export const showsHumanCheck = ({ siteKey = '', native = false, passed = false } = {}) =>
  !native && Boolean(siteKey) && !passed

export const humanBlocked = ({ ready = false, siteKey = '', native = false, passed = false,
  token = '' } = {}) => {
  if (!ready) return true
  if (!showsHumanCheck({ siteKey, native, passed })) return false
  return !token
}

export default showsHumanCheck
