import cx from '../../lib/cx'

export const UNIVERSITY = 'Shahjalal University of Science and Technology'

export const DEPARTMENT = 'Department of Computer Science and Engineering'

export default function IdentityBar({ right, className }) {
  return (
    <div
      className={cx(
        'flex flex-wrap justify-between gap-x-4 gap-y-1 bg-ink px-4 py-2.5 text-xs text-white/85 sm:px-6',
        className,
      )}
    >
      <span>
        <span className="font-semibold text-white">{UNIVERSITY}</span>
        <span className="hidden sm:inline"> · {DEPARTMENT}</span>
      </span>
      {right && <span className="shrink-0">{right}</span>}
    </div>
  )
}
