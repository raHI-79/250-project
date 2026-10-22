import { useEffect, useState } from 'react'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { useCountUp } from '../../lib/hooks'

const FIELDS = [
  { key: 'students', label: 'Active students' },
  { key: 'teachers', label: 'Teachers' },
  { key: 'courses', label: 'Courses' },
  { key: 'batches', label: 'Batches' },
]

function StatNumber({ value, label }) {
  const shown = useCountUp(value)
  return (
    <div className="border-l border-line px-4 py-6 first:border-l-0 sm:px-6 sm:py-8">
      <p className="label-caps">{label}</p>
      <p className="font-numeral mt-2 text-[34px] leading-none font-bold text-brand-600 sm:text-[38px]">
        {shown}
      </p>
    </div>
  )
}

export default function StatsStrip({ className }) {
  const [stats, setStats] = useState(null)
  useEffect(() => {
    let active = true
    api
      .get('/api/public/stats', { auth: false })
      .then((data) => {
        if (active) setStats(data || {})
      })
      .catch(() => null)
    return () => {
      active = false
    }
  }, [])
  return (
    <section
      className={cx('border-y border-line bg-surface', className)}
      aria-label="Department at a glance"
    >
      <div className="mx-auto grid max-w-6xl grid-cols-2 px-0 sm:px-6 lg:grid-cols-4">
        {FIELDS.map((field) => (
          <StatNumber
            key={field.key}
            label={field.label}
            value={Number(stats?.[field.key]) || 0}
          />
        ))}
      </div>
    </section>
  )
}
