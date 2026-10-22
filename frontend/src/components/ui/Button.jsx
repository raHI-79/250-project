import { Loader2 } from 'lucide-react'
import cx from '../../lib/cx'

const variants = {
  primary: 'bg-brand-600 text-white hover:bg-brand-800 disabled:bg-brand-300',
  secondary: 'border border-brand-600 bg-surface text-brand-600 hover:bg-brand-50 disabled:border-line-strong disabled:text-faint',
  danger: 'bg-accent-600 text-white hover:bg-accent-700 disabled:bg-accent-300',
  ghost: 'bg-transparent text-brand-600 hover:bg-brand-50 disabled:text-faint',
  ink: 'bg-ink text-white hover:bg-ink-soft disabled:bg-faint',
  light: 'bg-surface text-brand-600 hover:bg-brand-50 disabled:text-faint',
  outlineLight: 'border border-white/70 text-white hover:bg-white/10 disabled:border-white/30',
}

const sizes = {
  sm: 'h-9 px-3 text-sm sm:h-9',
  md: 'h-11 px-4 text-sm',
  lg: 'h-12 px-6 text-base',
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
  ...rest
}) {
  const native = Tag === 'button'
  return (
    <Tag
      type={native ? type : undefined}
      disabled={native ? disabled || loading : undefined}
      aria-disabled={!native && (disabled || loading) ? 'true' : undefined}
      className={cx(
        'inline-flex items-center justify-center gap-2 rounded-md font-semibold whitespace-nowrap transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 active:scale-[0.98] disabled:cursor-not-allowed',
        variants[variant] || variants.primary,
        sizes[size] || sizes.md,
        className,
      )}
      {...rest}
    >
      {loading && <Loader2 size={15} className="animate-spin" />}
      {children}
    </Tag>
  )
}
