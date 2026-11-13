import { paths } from './paths.js'

export const BANNER_CLASS = 'has-app-banner'

const clean = (pathname) => {
  const path = String(pathname || '').split('?')[0].replace(/\/+$/, '')
  return path || '/'
}

export const accountPaths = () =>
  [
    paths.studentLogin(),
    paths.studentRegister(),
    paths.studentForgotPassword(),
    paths.teacherLogin(),
    paths.teacherForgotPassword(),
    paths.teacherActivate(),
    paths.adminLogin(),
    paths.adminForgotPassword(),
    paths.adminInvite(),
    paths.controlLogin(),
    paths.controlForgotPassword(),
  ].map(clean)

export const isAccountPath = (pathname) => accountPaths().includes(clean(pathname))

export const bannerShows = ({ androidBrowser, dismissed, pathname }) =>
  Boolean(androidBrowser) && !dismissed && !isAccountPath(pathname)
