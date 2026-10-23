import cx from '../../lib/cx'

const TONES = {
  neutral: 'border-line bg-wash text-ink-soft',
  brand: 'border-brand-200 bg-brand-50 text-brand-700',
  gold: 'border-gold-200 bg-gold-50 text-gold-700',
  success: 'border-success-100 bg-success-50 text-success-700',
  warning: 'border-warning-100 bg-warning-50 text-warning-700',
  danger: 'border-danger-100 bg-danger-50 text-danger-700',
}

export default function Chip({
  as: Tag = 'span',
  icon: Icon,
  tone = 'neutral',
  selected = false,
  className,
  children,
  ...rest
}) {
  const interactive = Tag !== 'span'
  return (
    <Tag
      type={Tag === 'button' ? 'button' : undefined}
      aria-pressed={Tag === 'button' ? selected : undefined}
      className={cx(
        'inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-semibold whitespace-nowrap',
        selected ? 'border-brand-600 bg-brand-600 text-page' : TONES[tone] || TONES.neutral,
        interactive && 'min-h-9 transition-colors',
        interactive && !selected && 'hover:border-brand-300 hover:bg-brand-50',
        className,
      )}
      {...rest}
    >
      {Icon && <Icon size={13} className="shrink-0" />}
      {children}
    </Tag>
  )
}
