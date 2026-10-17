import { useId } from 'react'
import cx from '../../lib/cx'

const controlClass = (invalid) =>
  cx(
    'w-full rounded-lg border bg-white px-3 py-2 text-sm text-brand-950 transition-colors placeholder:text-brand-300 focus:outline-none focus:ring-2',
    invalid
      ? 'border-accent-400 focus:border-accent-500 focus:ring-accent-200'
      : 'border-brand-200 focus:border-brand-500 focus:ring-brand-100',
  )

export function Field({ label, error, hint, htmlFor, required = false, className, children }) {
  return (
    <div className={cx('flex flex-col gap-1.5', className)}>
      {label && (
        <label htmlFor={htmlFor} className="text-sm font-medium text-brand-800">
          {label}
          {required && <span className="ml-1 text-accent-600">*</span>}
        </label>
      )}
      {children}
      {error ? (
        <p className="animate-fade-in text-xs text-accent-700">{error}</p>
      ) : (
        hint && <p className="text-xs text-brand-400">{hint}</p>
      )}
    </div>
  )
}

export function Input({ label, error, hint, required, className, id, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  return (
    <Field
      label={label}
      error={error}
      hint={hint}
      required={required}
      htmlFor={fieldId}
      className={className}
    >
      <input id={fieldId} className={controlClass(Boolean(error))} {...rest} />
    </Field>
  )
}

export function Select({ label, error, hint, required, className, id, children, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  return (
    <Field
      label={label}
      error={error}
      hint={hint}
      required={required}
      htmlFor={fieldId}
      className={className}
    >
      <select id={fieldId} className={controlClass(Boolean(error))} {...rest}>
        {children}
      </select>
    </Field>
  )
}

export function Textarea({ label, error, hint, required, className, id, rows = 4, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  return (
    <Field
      label={label}
      error={error}
      hint={hint}
      required={required}
      htmlFor={fieldId}
      className={className}
    >
      <textarea
        id={fieldId}
        rows={rows}
        className={cx(controlClass(Boolean(error)), 'resize-y')}
        {...rest}
      />
    </Field>
  )
}

export default Field
