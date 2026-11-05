import { useEffect, useId, useRef, useState } from 'react'
import { CalendarDays } from 'lucide-react'
import cx from '../../lib/cx'
import { isNative } from '../../lib/native'
import { dateParts, formatDate, fromParts, isValidDate, WEEKDAYS } from '../../lib/time'
import { Field } from './Field'

const MONTH_NAMES = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
]

const EMPTY = { day: '', month: '', year: '' }

const digits = (value, length) => String(value ?? '').replace(/\D/g, '').slice(0, length)

const box = (dark) =>
  cx(
    'h-12 bg-transparent text-center text-base tabular-nums outline-none',
    dark ? 'text-white placeholder:text-white/40' : 'text-ink placeholder:text-faint',
  )

// DateField asks for the day, then the month, then the year, and hands back a
// YYYY-MM-DD string. The browser's own date input shows whatever order its locale
// prefers, which read 11/06/2026 for a date in November on an American locale, so the
// portal and the app use this field everywhere instead.
export default function DateField({
  label = 'Date',
  value,
  onChange,
  min,
  max,
  error,
  hint,
  required,
  disabled,
  tone,
  className,
  id,
  name,
}) {
  const dark = tone === 'dark'
  const generated = useId()
  const fieldId = id || generated
  const [parts, setParts] = useState(() => dateParts(value))
  const [focused, setFocused] = useState(false)
  const dayRef = useRef(null)
  const monthRef = useRef(null)
  const yearRef = useRef(null)
  const native = useRef(null)
  useEffect(() => {
    const incoming = dateParts(value)
    setParts((current) => {
      if (fromParts(current) === value) return current
      return incoming
    })
  }, [value])
  const rangeError = (stamp) => {
    if (min && stamp < min) return `Choose ${formatDate(min)} or later.`
    if (max && stamp > max) return `Choose ${formatDate(max)} or earlier.`
    return ''
  }
  const [complaint, setComplaint] = useState('')
  const settle = (next) => {
    setParts(next)
    const stamp = fromParts(next)
    if (!stamp) {
      if (!next.day && !next.month && !next.year) {
        setComplaint('')
        onChange?.('')
      }
      return
    }
    const wrong = rangeError(stamp)
    setComplaint(wrong)
    if (wrong) return
    onChange?.(stamp)
  }
  const change = (key, length, nextRef) => (event) => {
    const raw = digits(event.target.value, length)
    const next = { ...parts, [key]: raw }
    settle(next)
    if (raw.length === length && nextRef?.current) nextRef.current.focus()
  }
  const back = (key, previousRef) => (event) => {
    if (event.key !== 'Backspace' || parts[key] !== '' || !previousRef?.current) return
    event.preventDefault()
    previousRef.current.focus()
  }
  const openPicker = () => {
    const node = native.current
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
  }
  const stamp = fromParts(parts)
  const shown = complaint || error
  const weekday = isValidDate(stamp) ? WEEKDAYS[new Date(`${stamp}T00:00:00Z`).getUTCDay()] : ''
  const readable = isValidDate(stamp) ? formatDate(stamp) : ''
  return (
    <Field
      label={label}
      error={shown}
      hint={shown ? '' : readable || hint || 'Day / month / year'}
      htmlFor={`${fieldId}-day`}
      required={required}
      tone={tone}
      className={className}
    >
      <div
        className={cx(
          'field-skin relative flex items-center gap-1 rounded-input px-2',
          dark ? 'field-skin-dark bg-white/10' : 'bg-input',
          isNative() && focused ? 'border-2' : 'border',
          shown
            ? 'border-danger-500'
            : focused
              ? dark
                ? 'border-gold-300'
                : 'border-brand-600 shadow-[0_0_0_3px_var(--color-brand-100)]'
              : dark
                ? 'border-white/25'
                : 'border-input-line',
          disabled && 'opacity-60',
        )}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
      >
        <input
          ref={dayRef}
          id={`${fieldId}-day`}
          name={name ? `${name}_day` : undefined}
          inputMode="numeric"
          autoComplete="off"
          aria-label={`${label} day`}
          placeholder="DD"
          disabled={disabled}
          value={parts.day}
          onChange={change('day', 2, monthRef)}
          className={cx(box(dark), 'w-10')}
        />
        <span aria-hidden="true" className={dark ? 'text-white/40' : 'text-faint'}>
          /
        </span>
        <input
          ref={monthRef}
          id={`${fieldId}-month`}
          name={name ? `${name}_month` : undefined}
          inputMode="numeric"
          autoComplete="off"
          aria-label={`${label} month`}
          placeholder="MM"
          disabled={disabled}
          value={parts.month}
          onChange={change('month', 2, yearRef)}
          onKeyDown={back('month', dayRef)}
          className={cx(box(dark), 'w-10')}
        />
        <span aria-hidden="true" className={dark ? 'text-white/40' : 'text-faint'}>
          /
        </span>
        <input
          ref={yearRef}
          id={`${fieldId}-year`}
          name={name ? `${name}_year` : undefined}
          inputMode="numeric"
          autoComplete="off"
          aria-label={`${label} year`}
          placeholder="YYYY"
          disabled={disabled}
          value={parts.year}
          onChange={change('year', 4, null)}
          onKeyDown={back('year', monthRef)}
          className={cx(box(dark), 'w-16')}
        />
        <span className="ml-auto flex items-center gap-2 pr-0.5">
          {parts.month && MONTH_NAMES[Number(parts.month) - 1] && (
            <span className={cx('text-xs font-semibold', dark ? 'text-white/70' : 'text-muted')}>
              {MONTH_NAMES[Number(parts.month) - 1]}
              {weekday ? ` · ${weekday.slice(0, 3)}` : ''}
            </span>
          )}
          <button
            type="button"
            onClick={openPicker}
            disabled={disabled}
            aria-label={`Open a calendar for ${label}`}
            title="Open a calendar"
            className={cx(
              'press grid size-9 place-items-center rounded-full',
              dark ? 'text-gold-200' : 'text-brand-600',
            )}
          >
            <CalendarDays size={18} />
          </button>
        </span>
        <input
          ref={native}
          type="date"
          tabIndex={-1}
          aria-hidden="true"
          value={isValidDate(stamp) ? stamp : ''}
          min={min}
          max={max}
          disabled={disabled}
          onChange={(event) => {
            if (!event.target.value) return
            settle(dateParts(event.target.value))
          }}
          className="pointer-events-none absolute size-px opacity-0"
        />
      </div>
    </Field>
  )
}

export { EMPTY as EMPTY_DATE_PARTS }
