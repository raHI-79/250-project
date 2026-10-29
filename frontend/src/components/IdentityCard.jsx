import { useEffect, useState } from 'react'
import cx from '../lib/cx'
import { initialsOf } from '../lib/account'
import { assetURL } from '../lib/native'
import LogoBadge from './ui/LogoBadge'
import { DEPARTMENT, UNIVERSITY } from './ui/IdentityBar'

const QR_SIZE = 220

export default function IdentityCard({
  name,
  photoUrl,
  role,
  chips = [],
  rows = [],
  code,
  codeLabel = 'ID',
  footer,
  className,
}) {
  const [broken, setBroken] = useState(false)
  const [qr, setQr] = useState('')
  useEffect(() => setBroken(false), [photoUrl])
  useEffect(() => {
    if (!code) {
      setQr('')
      return undefined
    }
    let live = true
    import('qrcode')
      .then((module) =>
        module.default.toDataURL(String(code), {
          width: QR_SIZE,
          margin: 0,
          errorCorrectionLevel: 'M',
          color: { dark: '#141518ff', light: '#ffffffff' },
        }),
      )
      .then((url) => {
        if (live) setQr(url)
      })
      .catch(() => {
        if (live) setQr('')
      })
    return () => {
      live = false
    }
  }, [code])
  const showPhoto = Boolean(photoUrl) && !broken
  return (
    <div className={cx('glass-card overflow-hidden rounded-card', className)}>
      <div className="relative overflow-hidden bg-linear-to-br from-deep-700 to-deep-950 px-5 pt-5 pb-14 text-center">
        <div className="flex items-center justify-center gap-2">
          <LogoBadge size="sm" tone="light" className="size-8 rounded-lg p-1" />
          <p className="text-2xs font-bold tracking-[0.12em] text-white/85 uppercase">
            {UNIVERSITY}
          </p>
        </div>
        <p className="mt-1.5 text-2xs tracking-[0.06em] text-white/65 uppercase">{DEPARTMENT}</p>
      </div>
      <div className="relative z-10 -mt-11 flex flex-col items-center px-5">
        <div className="size-22 overflow-hidden rounded-full border-4 border-surface bg-wash">
          {showPhoto ? (
            <img
              src={assetURL(photoUrl)}
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
        <p className="font-display wrap-long mt-3 text-center text-section font-bold text-ink">
          {name}
        </p>
        {role && (
          <p className="mt-0.5 text-2xs font-bold tracking-[0.1em] text-brand-600 uppercase">
            {role}
          </p>
        )}
        {chips.length > 0 && (
          <div className="mt-2.5 flex flex-wrap justify-center gap-1.5">{chips}</div>
        )}
      </div>
      <dl className="mt-4 px-5">
        {rows
          .filter((row) => row.value)
          .map((row) => (
            <div
              key={row.label}
              className="flex items-baseline justify-between gap-4 border-b border-line py-2.5 last:border-b-0"
            >
              <dt className="text-2xs font-bold tracking-[0.08em] text-muted uppercase">
                {row.label}
              </dt>
              <dd className="wrap-long min-w-0 text-right text-sm font-medium text-ink">
                {row.value}
              </dd>
            </div>
          ))}
      </dl>
      {code && (
        <div className="mt-2 flex flex-col items-center border-t border-line px-5 py-5">
          {qr ? (
            <img
              src={qr}
              alt={`QR code of ${codeLabel} ${code}`}
              className="size-36 rounded-lg bg-white p-2"
            />
          ) : (
            <div className="size-36 animate-pulse rounded-lg bg-wash" />
          )}
          <p className="label-caps mt-3">{codeLabel}</p>
          <p className="font-numeral text-base font-bold tracking-[0.12em] text-ink">{code}</p>
        </div>
      )}
      {footer}
    </div>
  )
}
