import { Link } from 'react-router-dom'
import cx from '../lib/cx'
import { useCountUp } from '../lib/hooks'

export default function StatCard({ icon: Icon, label, value, to, hint }) {
  const shown = useCountUp(value)
  const body = (
    <div
      className={cx(
        'h-full border border-line bg-surface p-4',
        to && 'card-hover',
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <p className="label-caps">{label}</p>
        {Icon && <Icon size={16} className="shrink-0 text-brand-600" />}
      </div>
      <p className="font-numeral mt-2 text-[34px] leading-none font-bold text-brand-600">
        {shown}
      </p>
      {hint && <p className="mt-2 text-xs text-muted">{hint}</p>}
    </div>
  )
  if (!to) return body
  return (
    <Link to={to} className="block">
      {body}
    </Link>
  )
}
