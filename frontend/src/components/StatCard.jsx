import { Link } from 'react-router-dom'
import Card from './ui/Card'
import { useCountUp } from '../lib/hooks'

export default function StatCard({ icon: Icon, label, value, to, hint }) {
  const shown = useCountUp(value)
  const body = (
    <Card className="animate-fade-in-up h-full">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-brand-500">{label}</p>
          <p className="font-display mt-2 text-3xl font-semibold tabular-nums text-brand-900">
            {shown}
          </p>
          {hint && <p className="mt-1 text-xs text-brand-500">{hint}</p>}
        </div>
        <span className="flex size-11 shrink-0 items-center justify-center rounded-full bg-brand-50 text-brand-600">
          <Icon size={22} />
        </span>
      </div>
    </Card>
  )
  if (!to) return body
  return (
    <Link to={to} className="block">
      {body}
    </Link>
  )
}
