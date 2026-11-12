import cx from '../../lib/cx'
import BrandMark from '../ui/BrandMark'
import HeaderArt from '../ui/HeaderArt'
import { SUST_LOGO } from '../../lib/brand'
import { UNIVERSITY } from '../ui/IdentityBar'
import { APP_VERSION } from '../../lib/version'
import { appFooter } from '../../lib/authpanel'

const PANEL_MIN_HEIGHT = '20.625rem'

export default function AppAuthScreen({ title, subtitle, wide = false, className, children }) {
  return (
    <div className="relative flex min-h-dvh flex-col overflow-x-clip bg-header">
      <div
        style={{ minHeight: PANEL_MIN_HEIGHT }}
        className="relative isolate flex shrink-0 flex-col items-center justify-center gap-3 px-6 pt-[max(1.5rem,env(safe-area-inset-top,0px))] pb-14 text-center"
      >
        <HeaderArt variant="panel" />
        <span className="grid size-[5.75rem] shrink-0 place-items-center rounded-[1.625rem] bg-white p-3.5 shadow-[0_12px_28px_rgb(4_32_54/0.3)]">
          <BrandMark src={SUST_LOGO} alt={UNIVERSITY} className="size-full" />
        </span>
        <h1 className="font-display text-[1.625rem] leading-tight font-extrabold text-white">
          {title}
        </h1>
        {subtitle && <p className="text-[0.9375rem] leading-snug text-white/95">{subtitle}</p>}
      </div>
      <div className="relative -mt-10 flex flex-1 flex-col rounded-t-[1.75rem] bg-surface px-[1.375rem] pt-[1.625rem] pb-[max(1.625rem,env(safe-area-inset-bottom,0px))]">
        <div
          className={cx(
            'animate-fade-in-up mx-auto flex w-full flex-1 flex-col',
            wide ? 'max-w-lg' : 'max-w-sm',
            className,
          )}
        >
          {children}
          <p className="mt-auto pt-8 text-center text-2xs text-faint">{appFooter(APP_VERSION)}</p>
        </div>
      </div>
    </div>
  )
}
