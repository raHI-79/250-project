import { Link } from 'react-router-dom'
import paths from '../../lib/paths'
import { LogoBadge } from '../ui'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'

export default function SiteFooter() {
  return (
    <footer className="bg-night px-4 py-10 text-white/70 sm:px-6">
      <div className="mx-auto grid max-w-6xl gap-8 sm:grid-cols-2 lg:grid-cols-3">
        <div>
          <div className="flex items-center gap-3">
            <LogoBadge size="md" tone="light" />
            <p className="font-display text-sm font-semibold text-white">{UNIVERSITY}</p>
          </div>
          <p className="mt-3 text-xs leading-relaxed">{DEPARTMENT}</p>
        </div>
        <div>
          <p className="text-xs font-semibold tracking-[0.1em] text-white/50 uppercase">
            Address
          </p>
          <p className="mt-3 text-xs leading-relaxed">
            Academic Building A, University Avenue
            <br />
            Kumargaon, Sylhet 3114
            <br />
            Bangladesh
          </p>
        </div>
        <div>
          <p className="text-xs font-semibold tracking-[0.1em] text-white/50 uppercase">
            Sign in
          </p>
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
      </div>
      <p className="mx-auto mt-9 max-w-6xl border-t border-white/15 pt-5 text-xs text-white/45">
        For class and exam coordination inside the department only.
      </p>
    </footer>
  )
}
