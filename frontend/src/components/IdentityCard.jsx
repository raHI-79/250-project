import { useEffect, useState } from 'react'
import cx from '../lib/cx'
import { initialsOf } from '../lib/account'
import { DEPARTMENT, UNIVERSITY } from './ui/IdentityBar'

const BARS = [2, 1, 3, 1, 2, 4, 1, 2, 1, 3, 2, 1, 4, 1, 2, 3, 1, 2, 1, 3, 2, 4, 1, 2]

export default function IdentityCard({
  name,
  photoUrl,
  role,
  chips = [],
  rows = [],
  code,
  footer,
  className,
}) {
  const [broken, setBroken] = useState(false)
  useEffect(() => setBroken(false), [photoUrl])
  const showPhoto = Boolean(photoUrl) && !broken
  return (
    <div className={cx('border border-line bg-surface', className)}>
      <div className="bg-brand-600 px-4 py-3 text-center">
        <p className="text-xs font-semibold tracking-[0.1em] text-white/70 uppercase">
          {UNIVERSITY}
        </p>
        <p className="font-display mt-1 text-sm font-bold text-white">{DEPARTMENT}</p>
      </div>
      <div className="flex flex-col items-center border-b border-line px-5 py-5">
        <div className="size-24 overflow-hidden border border-line-strong bg-wash">
          {showPhoto ? (
            <img
              src={photoUrl}
              alt=""
              onError={() => setBroken(true)}
              className="size-full object-cover"
            />
          ) : (
            <span className="font-display grid size-full place-items-center text-2xl font-semibold text-brand-600 uppercase">
              {initialsOf(name)}
            </span>
          )}
        </div>
        <p className="font-display wrap-long mt-3 text-center text-xl font-bold text-ink">{name}</p>
        {role && (
          <p className="mt-1 text-xs font-semibold tracking-[0.1em] text-brand-600 uppercase">
            {role}
          </p>
        )}
        {chips.length > 0 && (
          <div className="mt-2.5 flex flex-wrap justify-center gap-1.5">{chips}</div>
        )}
      </div>
      <dl className="px-5 py-3">
        {rows
          .filter((row) => row.value)
          .map((row) => (
            <div
              key={row.label}
              className="flex items-baseline justify-between gap-4 border-b border-dashed border-line py-2.5 last:border-b-0"
            >
              <dt className="text-xs font-semibold tracking-[0.08em] text-muted uppercase">
                {row.label}
              </dt>
              <dd className="wrap-long min-w-0 text-right text-sm text-ink">{row.value}</dd>
            </div>
          ))}
      </dl>
      {code && (
        <div className="border-t border-line px-5 py-4 text-center">
          <div aria-hidden="true" className="flex h-8 items-end justify-center gap-[2px]">
            {BARS.map((width, position) => (
              <span
                key={`${width}-${position}`}
                style={{ width: `${width}px` }}
                className="h-full bg-ink"
              />
            ))}
          </div>
          <p className="font-numeral mt-2 text-xs tracking-[0.2em] text-muted">{code}</p>
        </div>
      )}
      {footer}
    </div>
  )
}
