import { CalendarCheck, GraduationCap, PieChart } from 'lucide-react'
import cx from '../../lib/cx'

function Stat({ icon: Icon, label, value, tone }) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-2.5 px-3 py-3 sm:px-4">
      <span
        aria-hidden="true"
        className={cx('grid size-10 shrink-0 place-items-center rounded-full', tone)}
      >
        <Icon size={18} strokeWidth={1.9} />
      </span>
      <span className="min-w-0">
        <span className="font-display block wrap-long text-lg leading-tight text-ink">{value}</span>
        <span className="block wrap-long text-2xs font-bold tracking-[0.06em] text-muted uppercase">
          {label}
        </span>
      </span>
    </div>
  )
}

export default function HomeStats({ classesToday, attendance, semester, className }) {
  return (
    <div
      className={cx(
        'surface-card relative -mt-4 flex sm:-mt-9 flex-col divide-y divide-line sm:flex-row sm:divide-x sm:divide-y-0',
        className,
      )}
    >
      <Stat
        icon={CalendarCheck}
        label="Classes today"
        value={Number(classesToday) || 0}
        tone="bg-tint-blue text-tint-blue-ink"
      />
      <Stat
        icon={PieChart}
        label="Attendance"
        value={attendance === null || attendance === undefined ? '—' : `${attendance} %`}
        tone="bg-tint-green text-tint-green-ink"
      />
      <Stat
        icon={GraduationCap}
        label="Semester"
        value={semester || '—'}
        tone="bg-tint-purple text-tint-purple-ink"
      />
    </div>
  )
}
