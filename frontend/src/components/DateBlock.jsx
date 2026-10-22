import cx from '../lib/cx'

const parse = (value) => {
  const date = value instanceof Date ? value : new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? null : date
}

export default function DateBlock({ value, className }) {
  const date = parse(value)
  if (!date) return null
  return (
    <div
      className={cx(
        'w-14 shrink-0 border border-ink text-center',
        className,
      )}
    >
      <p className="font-numeral pt-1 text-2xl leading-tight font-bold text-ink">
        {String(date.getDate()).padStart(2, '0')}
      </p>
      <p className="pb-1 text-xs font-semibold tracking-[0.08em] text-muted uppercase">
        {date.toLocaleDateString('en-GB', { month: 'short' })}
      </p>
      <p className="bg-ink py-0.5 text-xs font-semibold tracking-[0.08em] text-white uppercase">
        {date.toLocaleDateString('en-GB', { weekday: 'short' })}
      </p>
    </div>
  )
}
