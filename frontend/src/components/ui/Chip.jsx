import cx from '../../lib/cx'

const TONES = {
  neutral: 'border-line bg-wash text-ink-soft',
  brand: 'border-brand-100 bg-brand-50 text-brand-700',
  gold: 'border-gold-100 bg-gold-50 text-gold-600',
  success: 'border-success-100 bg-success-50 text-success-600',
  warning: 'border-warning-100 bg-warning-50 text-warning-600',
  danger: 'border-danger-100 bg-danger-50 text-danger-600',
  purple: 'border-purple-100 bg-purple-50 text-purple-600',
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
        'inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-bold whitespace-nowrap transition-colors duration-200',
        selected ? 'border-brand-600 bg-brand-600 text-page' : TONES[tone] || TONES.neutral,
        interactive && 'min-h-11',
        interactive && !selected && 'hover:border-brand-200 hover:bg-brand-50 hover:text-brand-700',
        className,
      )}
      {...rest}
    >
      {Icon && <Icon size={13} className="shrink-0" />}
      {children}
    </Tag>
  )
}
