import { useId, useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import cx from '../../lib/cx'

const shell = (invalid, tone, focused) =>
  cx(
    'relative rounded-xl border transition-all duration-200',
    tone === 'dark' ? 'bg-white/5' : 'glass-chip',
    invalid
      ? 'border-danger-500'
      : focused
        ? tone === 'dark'
          ? 'border-gold-300'
          : 'border-brand-600'
        : tone === 'dark'
          ? 'border-white/20'
          : 'border-line-strong',
    focused && !invalid && 'shadow-[0_0_0_3px_var(--color-brand-100)]',
    focused && invalid && 'shadow-[0_0_0_3px_var(--color-danger-100)]',
  )

const control = (tone, floating) =>
  cx(
    'w-full bg-transparent px-3.5 text-base outline-none',
    tone === 'dark' ? 'text-white placeholder:text-white/35' : 'text-ink placeholder:text-faint',
    floating ? 'h-12 pt-5 pb-1.5' : 'h-12',
  )

const labelClass = (tone, floated, invalid) =>
  cx(
    'pointer-events-none absolute left-3.5 origin-left transition-all duration-200 motion-reduce:transition-none',
    floated ? 'top-1.5 text-2xs font-semibold' : 'top-1/2 -translate-y-1/2 text-base',
    invalid
      ? 'text-danger-600'
      : tone === 'dark'
        ? 'text-white/65'
        : floated
          ? 'text-brand-600'
          : 'text-muted',
  )

export function Field({ label, error, hint, htmlFor, required = false, tone, className, children }) {
  const dark = tone === 'dark'
  return (
    <div className={cx('flex flex-col gap-1.5', className)}>
      {label && (
        <label
          htmlFor={htmlFor}
          className={cx('text-sm font-semibold', dark ? 'text-white/80' : 'text-ink-soft')}
        >
          {label}
          {required && <span className={cx('ml-1', dark ? 'text-sand' : 'text-brand-600')}>*</span>}
        </label>
      )}
      {children}
      {error ? (
        <p className="animate-fade-in text-sm text-danger-600">{error}</p>
      ) : (
        hint && <p className={cx('text-sm', dark ? 'text-white/60' : 'text-muted')}>{hint}</p>
      )}
    </div>
  )
}

function Floating({ label, error, hint, required, tone, className, id, children }) {
  const dark = tone === 'dark'
  return (
    <div className={cx('flex flex-col gap-1.5', className)}>
      {children}
      {error ? (
        <p className="animate-fade-in px-1 text-sm text-danger-600">{error}</p>
      ) : (
        hint && (
          <p className={cx('px-1 text-sm', dark ? 'text-white/60' : 'text-muted')}>{hint}</p>
        )
      )}
      {label && (
        <label htmlFor={id} className="sr-only">
          {label}
          {required ? ' (required)' : ''}
        </label>
      )}
    </div>
  )
}

const useFloating = (value, defaultValue) => {
  const [focused, setFocused] = useState(false)
  const filled = value !== undefined ? String(value ?? '') !== '' : Boolean(defaultValue)
  return { focused, setFocused, floated: focused || filled }
}

export function Input({ label, error, hint, required, tone, className, id, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  const { focused, setFocused, floated } = useFloating(rest.value, rest.defaultValue)
  return (
    <Floating
      label={label}
      error={error}
      hint={hint}
      required={required}
      tone={tone}
      className={className}
      id={fieldId}
    >
      <div className={shell(Boolean(error), tone, focused)}>
        <input
          id={fieldId}
          aria-label={label}
          required={required}
          className={cx(
            control(tone, Boolean(label)),
            !floated && 'placeholder:text-transparent',
          )}
          {...rest}
          onFocus={(event) => {
            setFocused(true)
            rest.onFocus?.(event)
          }}
          onBlur={(event) => {
            setFocused(false)
            rest.onBlur?.(event)
          }}
        />
        {label && (
          <span className={labelClass(tone, floated, Boolean(error))}>
            {label}
            {required && <span className="ml-0.5">*</span>}
          </span>
        )}
      </div>
    </Floating>
  )
}

export function PasswordInput({ className, ...rest }) {
  const [shown, setShown] = useState(false)
  const dark = rest.tone === 'dark'
  return (
    <div className={cx('relative', className)}>
      <Input {...rest} type={shown ? 'text' : 'password'} className="[&_input]:pr-12" />
      <button
        type="button"
        onClick={() => setShown((current) => !current)}
        aria-label={shown ? 'Hide password' : 'Show password'}
        aria-pressed={shown}
        title={shown ? 'Hide password' : 'Show password'}
        className={cx(
          'absolute top-0 right-0 grid h-12 w-12 place-items-center rounded-xl transition-colors',
          dark ? 'text-white/60 hover:text-white' : 'text-muted hover:text-ink',
        )}
      >
        {shown ? <EyeOff size={18} /> : <Eye size={18} />}
      </button>
    </div>
  )
}

export function Select({ label, error, hint, required, tone, className, id, children, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  const { focused, setFocused } = useFloating(rest.value, rest.defaultValue)
  return (
    <Floating
      label={label}
      error={error}
      hint={hint}
      required={required}
      tone={tone}
      className={className}
      id={fieldId}
    >
      <div className={shell(Boolean(error), tone, focused)}>
        <select
          id={fieldId}
          aria-label={label}
          required={required}
          className={control(tone, Boolean(label))}
          {...rest}
          onFocus={(event) => {
            setFocused(true)
            rest.onFocus?.(event)
          }}
          onBlur={(event) => {
            setFocused(false)
            rest.onBlur?.(event)
          }}
        >
          {children}
        </select>
        {label && (
          <span className={labelClass(tone, true, Boolean(error))}>
            {label}
            {required && <span className="ml-0.5">*</span>}
          </span>
        )}
      </div>
    </Floating>
  )
}

export function Textarea({ label, error, hint, required, tone, className, id, rows = 4, ...rest }) {
  const generated = useId()
  const fieldId = id || generated
  const { focused, setFocused, floated } = useFloating(rest.value, rest.defaultValue)
  return (
    <Floating
      label={label}
      error={error}
      hint={hint}
      required={required}
      tone={tone}
      className={className}
      id={fieldId}
    >
      <div className={shell(Boolean(error), tone, focused)}>
        <textarea
          id={fieldId}
          rows={rows}
          aria-label={label}
          required={required}
          className={cx(
            'w-full resize-y bg-transparent px-3.5 leading-relaxed outline-none',
            tone === 'dark'
              ? 'text-white placeholder:text-white/35'
              : 'text-ink placeholder:text-faint',
            label ? 'pt-6 pb-2.5' : 'py-2.5',
            !floated && 'placeholder:text-transparent',
          )}
          {...rest}
          onFocus={(event) => {
            setFocused(true)
            rest.onFocus?.(event)
          }}
          onBlur={(event) => {
            setFocused(false)
            rest.onBlur?.(event)
          }}
        />
        {label && (
          <span
            className={cx(
              'pointer-events-none absolute left-3.5 origin-left transition-all duration-200 motion-reduce:transition-none',
              floated ? 'top-2 text-2xs font-semibold' : 'top-3.5 text-base',
              error
                ? 'text-danger-600'
                : tone === 'dark'
                  ? 'text-white/65'
                  : floated
                    ? 'text-brand-600'
                    : 'text-muted',
            )}
          >
            {label}
            {required && <span className="ml-0.5">*</span>}
          </span>
        )}
      </div>
    </Floating>
  )
}

export default Field
