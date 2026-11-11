import { CalendarCheck, CalendarOff } from 'lucide-react'
import cx from '../../lib/cx'

export default function DayKindChip({ kind, className }) {
  if (!kind) return null
  const working = Boolean(kind.working_day)
  const Icon = working ? CalendarCheck : CalendarOff
  return (
    <p
      className={cx(
        'inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-sm font-semibold',
        working ? 'bg-success-50 text-success-600' : 'bg-warning-50 text-warning-600',
        className,
      )}
    >
      <Icon size={15} aria-hidden="true" />
      {kind.label}
    </p>
  )
}
