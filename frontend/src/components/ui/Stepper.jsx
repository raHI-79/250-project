import { Check } from 'lucide-react'
import cx from '../../lib/cx'

export default function Stepper({ steps, current, className }) {
  return (
    <ol className={cx('flex items-start gap-1', className)}>
      {steps.map((label, position) => {
        const number = position + 1
        const done = current > number
        const active = current === number
        return (
          <li key={label} className="flex flex-1 flex-col items-center gap-1.5 text-center">
            <span
              className={cx(
                'inline-flex size-7 items-center justify-center rounded-full text-xs font-semibold transition-colors',
                done || active ? 'bg-brand-600 text-white' : 'bg-brand-100 text-brand-500',
              )}
            >
              {done ? <Check size={14} /> : number}
            </span>
            <span
              className={cx(
                'text-[11px] leading-tight sm:text-xs',
                done || active ? 'font-medium text-brand-700' : 'text-brand-400',
              )}
            >
              {label}
            </span>
          </li>
        )
      })}
    </ol>
  )
}
