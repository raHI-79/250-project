import { Link } from 'react-router-dom'
import { assetURL } from '../../lib/native'
import paths from '../../lib/paths'
import { isSafeImage, withDefaults } from '../../lib/siteContent'
import { IdentityBar } from '../ui'
import { SafeLink } from './RichText'

export default function PublicHeader({ content, preview = false }) {
  const header = withDefaults(content).header
  return (
    <>
      {preview && (
        <p
          role="status"
          data-site-preview
          className="bg-warning-600 px-4 py-2 text-center text-xs font-semibold text-white"
        >
          Preview of the draft. Visitors still see the published website.
        </p>
      )}
      <IdentityBar right="Class and exam coordination" />
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <Link to={paths.root()} className="flex min-w-0 items-center gap-3">
            {isSafeImage(header.logo_url) && (
              <img
                src={assetURL(header.logo_url)}
                alt=""
                className="size-11 shrink-0 rounded-full bg-white object-contain"
              />
            )}
            <span className="min-w-0">
              <span data-site-name className="font-display block text-base font-semibold text-ink">
                {header.site_name}
              </span>
              <span className="block truncate text-xs text-muted">{header.tagline}</span>
            </span>
          </Link>
          <nav className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm font-semibold">
            {header.menu.map((entry) => (
              <SafeLink
                key={`${entry.label}-${entry.url}`}
                to={entry.url}
                className="text-ink-soft hover:text-brand-600"
              >
                {entry.label}
              </SafeLink>
            ))}
            <Link to={paths.studentLogin()} className="text-ink-soft hover:text-brand-600">
              Student
            </Link>
            <Link to={paths.teacherLogin()} className="text-ink-soft hover:text-brand-600">
              Teacher
            </Link>
            <Link
              to={paths.studentRegister()}
              className="rounded-xl border border-brand-600 px-3 py-1.5 text-brand-600 hover:bg-brand-50"
            >
              Register
            </Link>
          </nav>
        </div>
      </header>
    </>
  )
}
