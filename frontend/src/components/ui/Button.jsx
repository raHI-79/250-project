import { useCallback, useRef, useState } from 'react'
import { Loader2 } from 'lucide-react'
import cx from '../../lib/cx'

const variants = {
  primary:
    'bg-brand-600 text-page shadow-[var(--shadow-card)] hover:bg-brand-800 disabled:bg-brand-200 disabled:text-brand-700 disabled:shadow-none',
  secondary:
    'border border-line-strong bg-surface text-ink hover:border-brand-200 hover:bg-brand-50 hover:text-brand-700 disabled:text-faint',
  danger:
    'bg-danger-600 text-page shadow-[var(--shadow-card)] hover:bg-danger-700 disabled:bg-danger-200 disabled:text-danger-700 disabled:shadow-none',
  ghost: 'bg-transparent text-brand-600 hover:bg-brand-50 disabled:text-faint',
  ink: 'bg-ink text-page hover:bg-ink-soft disabled:bg-faint',
  light: 'bg-surface text-brand-600 hover:bg-brand-50 disabled:text-faint',
  outlineLight: 'border border-white/70 text-white hover:bg-white/10 disabled:border-white/30',
}

const sizes = {
  sm: 'h-11 px-3.5 text-sm',
  md: 'h-12 px-5 text-sm',
  lg: 'h-13 px-6 text-base',
}

export default function Button({
  as: Tag = 'button',
  variant = 'primary',
  size = 'md',
  loading = false,
  disabled = false,
  type = 'button',
  className,
  children,
  onPointerDown,
  ...rest
}) {
  const native = Tag === 'button'
  const [ripples, setRipples] = useState([])
  const counter = useRef(0)
  const start = useCallback(
    (event) => {
      onPointerDown?.(event)
      if (disabled || loading) return
      const box = event.currentTarget.getBoundingClientRect()
      counter.current += 1
      const id = counter.current
      const size = Math.max(box.width, box.height)
      setRipples((current) => [
        ...current,
        { id, size, x: event.clientX - box.left - size / 2, y: event.clientY - box.top - size / 2 },
      ])
      window.setTimeout(() => setRipples((current) => current.filter((r) => r.id !== id)), 600)
    },
    [disabled, loading, onPointerDown],
  )
  return (
    <Tag
      type={native ? type : undefined}
      disabled={native ? disabled || loading : undefined}
      aria-disabled={!native && (disabled || loading) ? 'true' : undefined}
      aria-busy={loading || undefined}
      onPointerDown={start}
      className={cx(
        'press relative inline-flex items-center justify-center overflow-hidden rounded-input font-bold whitespace-nowrap focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:active:scale-100',
        variants[variant] || variants.primary,
        sizes[size] || sizes.md,
        className,
      )}
      {...rest}
    >
      {ripples.map((ripple) => (
        <span
          key={ripple.id}
          aria-hidden="true"
          style={{ width: ripple.size, height: ripple.size, left: ripple.x, top: ripple.y }}
          className="animate-ripple pointer-events-none absolute rounded-full bg-current opacity-25 motion-reduce:hidden"
        />
      ))}
      <span className="relative inline-flex items-center justify-center gap-2">
        {loading && <Loader2 size={15} className="animate-spin" />}
        {children}
      </span>
    </Tag>
  )
}
