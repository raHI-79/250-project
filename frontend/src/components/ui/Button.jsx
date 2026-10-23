import { Loader2 } from 'lucide-react'
import cx from '../../lib/cx'

const variants = {
  primary: 'bg-brand-600 text-page hover:bg-brand-700 disabled:bg-brand-300',
  secondary:
    'border border-line-strong bg-surface text-ink hover:border-brand-300 hover:bg-brand-50 hover:text-brand-700 disabled:border-line disabled:text-faint',
  danger: 'bg-danger-600 text-page hover:bg-danger-700 disabled:bg-danger-300',
  ghost: 'bg-transparent text-brand-600 hover:bg-brand-50 disabled:text-faint',
  ink: 'bg-ink text-page hover:bg-ink-soft disabled:bg-faint',
  light: 'bg-surface text-brand-600 hover:bg-brand-50 disabled:text-faint',
  outlineLight: 'border border-white/70 text-white hover:bg-white/10 disabled:border-white/30',
}

const sizes = {
  sm: 'h-10 px-3.5 text-sm',
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
  ...rest
}) {
  const native = Tag === 'button'
  return (
    <Tag
      type={native ? type : undefined}
      disabled={native ? disabled || loading : undefined}
      aria-disabled={!native && (disabled || loading) ? 'true' : undefined}
      className={cx(
        'inline-flex items-center justify-center gap-2 rounded-xl font-semibold whitespace-nowrap transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 active:scale-[0.98] disabled:cursor-not-allowed',
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
