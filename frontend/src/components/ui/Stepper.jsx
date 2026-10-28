import { Check } from 'lucide-react'
import cx from '../../lib/cx'

export default function Stepper({ steps, current, className }) {
  return (
    <ol className={cx('glass-card flex overflow-x-auto rounded-card', className)}>
      {steps.map((label, index) => {
        const step = index + 1
        const done = step < current
        const active = step === current
        return (
          <li
            key={label}
            className={cx(
              'min-w-[7.5rem] flex-1 border-r border-line px-4 py-3 last:border-r-0',
              active && 'shadow-[inset_0_-3px_0_var(--color-brand-600)]',
            )}
          >
            <p className="text-xs text-muted">Step {step}</p>
            <p
              className={cx(
                'font-display mt-0.5 flex items-center gap-1.5 text-sm font-semibold',
                done ? 'text-success-700' : active ? 'text-ink' : 'text-faint',
              )}
            >
              {label}
              {done && <Check size={13} />}
            </p>
          </li>
        )
      })}
    </ol>
  )
}
