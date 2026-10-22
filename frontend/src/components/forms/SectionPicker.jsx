import { useEffect, useMemo, useState } from 'react'
import { Field, Input } from '../ui'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { useDebounced } from '../../lib/hooks'

const BATCH_SHAPE = /^[A-Z]{2,10}-\d{2}$/

const OTHER = 'other'

export default function SectionPicker({ batch, value, onChange, error, className }) {
  const settled = useDebounced((batch || '').trim().toUpperCase(), 400)
  const [suggested, setSuggested] = useState([])
  const [mode, setMode] = useState('')
  useEffect(() => {
    if (!BATCH_SHAPE.test(settled)) {
      setSuggested([])
      return undefined
    }
    let live = true
    api
      .get('/api/auth/sections', { query: { batch: settled }, auth: false })
      .then((data) => {
        if (live) setSuggested(Array.isArray(data?.items) ? data.items : [])
      })
      .catch(() => {
        if (live) setSuggested([])
      })
    return () => {
      live = false
    }
  }, [settled])
  const options = useMemo(() => {
    const seen = new Set(suggested)
    if (value && !seen.has(value)) return [...suggested, value]
    return suggested
  }, [suggested, value])
  const typing = mode === OTHER || options.length === 0
  return (
    <Field
      label="Section"
      error={error}
      hint="One capital letter, for example A"
      required
      className={className}
    >
      {options.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {options.map((section) => (
            <button
              key={section}
              type="button"
              onClick={() => {
                setMode('')
                onChange(section)
              }}
              className={cx(
                'tap-target min-w-11 rounded-md border px-3 text-sm font-semibold transition-colors',
                value === section && mode !== OTHER
                  ? 'border-brand-600 bg-brand-600 text-white'
                  : 'border-line-strong bg-surface text-ink-soft hover:bg-wash',
              )}
            >
              {section}
            </button>
          ))}
          <button
            type="button"
            onClick={() => {
              setMode(OTHER)
              onChange('')
            }}
            className={cx(
              'tap-target rounded-md border px-3 text-sm font-semibold transition-colors',
              mode === OTHER
                ? 'border-brand-600 bg-brand-600 text-white'
                : 'border-line-strong bg-surface text-ink-soft hover:bg-wash',
            )}
          >
            Other
          </button>
        </div>
      )}
      {typing && (
        <Input
          aria-label="Section"
          value={value}
          onChange={(event) => onChange(event.target.value.toUpperCase().slice(0, 1))}
          maxLength={1}
          placeholder="A"
          className={options.length > 0 ? 'mt-2' : ''}
        />
      )}
    </Field>
  )
}
