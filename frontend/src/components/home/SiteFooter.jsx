import { Link } from 'react-router-dom'
import paths from '../../lib/paths'
import { footerParts, withDefaults } from '../../lib/siteContent'
import { LogoBadge } from '../ui'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import DeveloperCredit from '../site/DeveloperCredit'
import { SafeLink } from '../site/RichText'

const heading = 'text-xs font-semibold tracking-[0.1em] text-white/50 uppercase'

function LinkList({ title, links }) {
  if (links.length === 0) return null
  return (
    <div>
      <p className={heading}>{title}</p>
      <ul className="mt-3 space-y-2 text-xs">
        {links.map((entry) => (
          <li key={`${entry.label}-${entry.url}`}>
            <SafeLink to={entry.url} className="hover:text-white">
              {entry.label}
            </SafeLink>
          </li>
        ))}
      </ul>
    </div>
  )
}

export default function SiteFooter({ content }) {
  const { shown, footer } = footerParts(content)
  const header = withDefaults(content).header
  const contact = [footer.phone, footer.email, footer.office_hours].filter(Boolean)
  return (
    <footer data-site-footer className="bg-night px-4 py-10 text-white/70 sm:px-6">
      {shown && (
        <div className="mx-auto grid max-w-6xl gap-8 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <div className="flex items-center gap-3">
              <LogoBadge size="md" tone="light" />
              <p className="font-display text-sm font-semibold text-white">{UNIVERSITY}</p>
            </div>
            <p className="mt-3 text-xs leading-relaxed">{header.tagline || DEPARTMENT}</p>
          </div>
          {(footer.address || contact.length > 0) && (
            <div>
              <p className={heading}>Address</p>
              {footer.address && (
                <p className="mt-3 text-xs leading-relaxed whitespace-pre-line">{footer.address}</p>
              )}
              {contact.length > 0 && (
                <ul className="mt-3 space-y-1 text-xs">
                  {footer.phone && <li>Phone: {footer.phone}</li>}
                  {footer.email && <li>Email: {footer.email}</li>}
                  {footer.office_hours && <li>Office hours: {footer.office_hours}</li>}
                </ul>
              )}
            </div>
          )}
          <div>
            <p className={heading}>Sign in</p>
            <ul className="mt-3 space-y-2 text-xs">
              <li>
                <Link to={paths.studentLogin()} className="hover:text-white">
                  Student portal
                </Link>
              </li>
              <li>
                <Link to={paths.studentRegister()} className="hover:text-white">
                  Student registration
                </Link>
              </li>
              <li>
                <Link to={paths.teacherLogin()} className="hover:text-white">
                  Teacher portal
                </Link>
              </li>
            </ul>
          </div>
          <div className="flex flex-col gap-6">
            <LinkList title="Quick links" links={footer.quick_links} />
            <LinkList title="Follow" links={footer.social_links} />
          </div>
        </div>
      )}
      <div
        className={
          shown
            ? 'mx-auto mt-9 flex max-w-6xl flex-wrap items-center justify-between gap-3 border-t border-white/15 pt-5'
            : 'mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3'
        }
      >
        <p className="text-xs text-white/45">{shown ? footer.note : ''}</p>
        <DeveloperCredit />
      </div>
    </footer>
  )
}
