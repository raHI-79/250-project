import { Link } from 'react-router-dom'
import cx from '../../lib/cx'
import paths from '../../lib/paths'
import LogoBadge from '../ui/LogoBadge'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import CampusBackdrop from './CampusBackdrop'

const TONES = {
  brand: {
    overlay:
      'bg-linear-to-b from-maroon-950/45 via-maroon-900/25 to-maroon-950/55',
    card: 'glass-auth',
  },
  dark: {
    overlay: 'bg-linear-to-b from-black/70 via-maroon-950/55 to-black/75',
    card: 'glass-auth',
  },
}

export default function AuthScreen({
  images,
  rotateAfter,
  overlay,
  tone = 'brand',
  note,
  wide = false,
  className,
  children,
}) {
  const skin = TONES[tone] || TONES.brand
  return (
    <div className="relative flex min-h-dvh flex-col">
      <CampusBackdrop
        images={images}
        rotateAfter={rotateAfter}
        overlay={overlay || skin.overlay}
        className="fixed"
      />
      <div className="relative flex min-h-dvh flex-col px-4 pt-[max(1rem,env(safe-area-inset-top,0px))] pb-[max(1rem,env(safe-area-inset-bottom,0px))] sm:px-6">
        <Link to={paths.root()} className="flex items-center gap-2.5 self-start">
          <LogoBadge size="sm" tone="light" />
          <span className="min-w-0">
            <span className="font-display block truncate text-sm font-semibold text-white">
              {UNIVERSITY}
            </span>
            <span className="block truncate text-2xs text-white/70">{DEPARTMENT}</span>
          </span>
        </Link>
        <div className="flex flex-1 items-center justify-center py-6">
          <div
            className={cx(
              'animate-fade-in-up w-full rounded-[1.5rem] p-6 sm:p-7',
              skin.card,
              wide ? 'max-w-2xl' : 'max-w-sm',
              className,
            )}
          >
            {children}
          </div>
        </div>
        {note && (
          <p className="mx-auto max-w-md text-center text-xs leading-relaxed text-white/80">
            {note}
          </p>
        )}
      </div>
    </div>
  )
}
