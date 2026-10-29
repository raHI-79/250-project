import cx from '../../lib/cx'
import { buildLabel } from '../../lib/version'

export default function AppAuthScreen({ wide = false, className, children }) {
  return (
    <div className="relative flex min-h-dvh flex-col overflow-x-hidden bg-page">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-0 top-0 h-56 bg-linear-to-b from-brand-100 via-brand-50 to-transparent"
      />
      <div className="relative flex min-h-dvh flex-col px-5 pt-[max(1.5rem,env(safe-area-inset-top,0px))] pb-[max(1rem,env(safe-area-inset-bottom,0px))]">
        <div className="flex flex-1 flex-col justify-center py-4">
          <div
            className={cx(
              'animate-fade-in-up mx-auto w-full',
              wide ? 'max-w-lg' : 'max-w-sm',
              className,
            )}
          >
            {children}
          </div>
        </div>
        <p className="pt-4 text-center text-2xs text-faint">{buildLabel()}</p>
      </div>
    </div>
  )
}
