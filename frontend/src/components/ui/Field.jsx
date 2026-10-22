import { useId } from 'react'
import cx from '../../lib/cx'

const controlClass = (invalid, tone) =>
  cx(
    'h-11 w-full rounded-md border px-3 text-base transition-colors focus:outline-none focus:ring-3',
    tone === 'dark'
      ? 'bg-white/5 text-white placeholder:text-white/35'
      : 'bg-surface text-ink placeholder:text-faint',
    invalid
      ? 'border-accent-500 focus:border-accent-600 focus:ring-accent-100'
      : tone === 'dark'
        ? 'border-white/20 focus:border-gold-300 focus:ring-white/10'
        : 'border-line-strong focus:border-brand-600 focus:ring-brand-100',
  )

export function Field({
  label,
  error,
  hint,
  htmlFor,
  required = false,
  tone,
  className,
  children,
}) {
  const dark = tone === 'dark'
  return (
    <div className={cx('flex flex-col gap-1.5', className)}>
      {label && (
        <label
          htmlFor={htmlFor}
          className={cx('text-sm font-semibold', dark ? 'text-white/80' : 'text-ink-soft')}
        >
          {label}
          {required && <span className={cx('ml-1', dark ? 'text-gold-300' : 'text-brand-600')}>*</span>}
        </label>
      )}
      {children}
      {error ? (
        <p className="animate-fade-in text-sm text-accent-600">{error}</p>
      ) : (
        hint && <p className={cx('text-sm', dark ? 'text-white/60' : 'text-muted')}>{hint}</p>
      )}
    </div>
  )
}

export function Input({ label, error, hint, required, tone, className, id, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  return (
    <Field
      label={label}
      error={error}
      hint={hint}
      required={required}
      htmlFor={fieldId}
      tone={tone}
      className={className}
    >
      <input id={fieldId} className={controlClass(Boolean(error), tone)} {...rest} />
    </Field>
  )
}

export function Select({ label, error, hint, required, tone, className, id, children, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  return (
    <Field
      label={label}
      error={error}
      hint={hint}
      required={required}
      htmlFor={fieldId}
      tone={tone}
      className={className}
    >
      <select id={fieldId} className={controlClass(Boolean(error), tone)} {...rest}>
        {children}
      </select>
    </Field>
  )
}

export function Textarea({ label, error, hint, required, tone, className, id, rows = 4, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  return (
    <Field
      label={label}
      error={error}
      hint={hint}
      required={required}
      htmlFor={fieldId}
      tone={tone}
      className={className}
    >
      <textarea
        id={fieldId}
        rows={rows}
        className={cx(controlClass(Boolean(error), tone), 'h-auto resize-y py-2.5 leading-relaxed')}
        {...rest}
      />
    </Field>
  )
}

export default Field
