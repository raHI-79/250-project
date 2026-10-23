import { Link } from 'react-router-dom'
import cx from '../../lib/cx'
import paths from '../../lib/paths'
import Crest from '../ui/Crest'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import CampusBackdrop from './CampusBackdrop'

const TONES = {
  brand: {
    page: 'bg-paper',
    bar: 'bg-night text-white/75',
    barName: 'text-white',
    panel: 'bg-surface',
    overlay: 'bg-linear-to-br from-maroon-950/90 via-maroon-900/70 to-maroon-700/35',
  },
  dark: {
    page: 'bg-night',
    bar: 'bg-black/70 text-white/60',
    barName: 'text-white/90',
    panel: 'bg-night',
    overlay: 'bg-linear-to-br from-black/92 via-maroon-950/88 to-maroon-900/70',
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
    <div className={cx('grid min-h-dvh lg:grid-cols-[1.1fr_1fr]', skin.page)}>
      <aside className="relative hidden overflow-hidden lg:block">
        <CampusBackdrop
          images={images}
          rotateAfter={rotateAfter}
          overlay={overlay || skin.overlay}
        />
        <div className="relative flex h-full flex-col justify-between p-8 xl:p-10">
          <Link to={paths.root()} className="flex items-center gap-3">
            <Crest tone="light" />
            <span>
              <span className="font-display block text-sm font-semibold text-white">
                {UNIVERSITY}
              </span>
              <span className="block text-xs text-white/65">{DEPARTMENT}</span>
            </span>
          </Link>
          <div>
            <p className="text-xs font-semibold tracking-[0.14em] text-sand uppercase">
              Class and exam coordination
            </p>
            <p className="font-display mt-3 max-w-md text-2xl leading-snug font-semibold text-white xl:text-3xl">
              {note || 'One record for the routine, the reschedules, the exams and the notices.'}
            </p>
          </div>
          <p className="text-xs text-white/55">Kumargaon, Sylhet 3114, Bangladesh</p>
        </div>
      </aside>
      <main className={cx('flex min-h-dvh flex-col', skin.panel)}>
        <div
          className={cx(
            'flex items-center gap-3 px-4 pt-[max(0.625rem,env(safe-area-inset-top,0px))] pb-2.5 text-xs sm:px-6 lg:hidden',
            skin.bar,
          )}
        >
          <span className={cx('font-semibold', skin.barName)}>SUST CSE</span>
          <span className="truncate">{DEPARTMENT}</span>
        </div>
        <div className="flex flex-1 items-center justify-center px-4 py-10 sm:px-8">
          <div
            className={cx(
              'animate-fade-in-up w-full',
              wide ? 'max-w-2xl' : 'max-w-sm',
              className,
            )}
          >
            {children}
          </div>
        </div>
      </main>
    </div>
  )
}
