import { useId, useRef } from 'react'
import { CalendarDays } from 'lucide-react'
import cx from '../lib/cx'

export default function DateButton({ value, onChange, min, max, label = 'Pick a date', className }) {
  const id = useId()
  const field = useRef(null)
  const open = () => {
    const node = field.current
    if (!node) return
    if (typeof node.showPicker === 'function') {
      try {
        node.showPicker()
        return
      } catch {
        node.focus()
        return
      }
    }
    node.focus()
    node.click()
  }
  return (
    <div className={cx('relative', className)}>
      <button
        type="button"
        onClick={open}
        aria-label={label}
        title={label}
        className="press grid size-12 place-items-center rounded-full bg-white text-brand-700 shadow-[var(--shadow-card)]"
      >
        <CalendarDays size={20} />
      </button>
      <label htmlFor={id} className="sr-only">
        {label}
      </label>
      <input
        ref={field}
        id={id}
        type="date"
        value={value || ''}
        min={min}
        max={max}
        onChange={(event) => {
          if (event.target.value) onChange(event.target.value)
        }}
        className="pointer-events-none absolute inset-x-0 bottom-0 size-px opacity-0"
      />
    </div>
  )
}
