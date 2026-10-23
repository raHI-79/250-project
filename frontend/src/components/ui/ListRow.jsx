import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import cx from '../../lib/cx'

const TONES = {
  neutral: 'bg-wash text-brand-600',
  brand: 'bg-brand-50 text-brand-600',
  gold: 'bg-gold-50 text-gold-700',
  danger: 'bg-danger-50 text-danger-600',
}

export default function ListRow({
  icon: Icon,
  iconTone = 'neutral',
  leading,
  title,
  subtitle,
  value,
  trailing,
  chevron,
  to,
  onClick,
  danger = false,
  className,
  ...rest
}) {
  const Tag = to ? Link : onClick ? 'button' : 'div'
  const tappable = Boolean(to || onClick)
  const showChevron = chevron === undefined ? tappable : chevron
  return (
    <Tag
      to={to}
      onClick={onClick}
      type={Tag === 'button' ? 'button' : undefined}
      className={cx(
        'flex w-full min-h-12 items-center gap-3 px-4 py-3 text-left',
        tappable && 'transition-colors hover:bg-wash active:bg-wash',
        className,
      )}
      {...rest}
    >
      {leading}
      {!leading && Icon && (
        <span
          className={cx(
            'grid size-9 shrink-0 place-items-center rounded-full',
            danger ? TONES.danger : TONES[iconTone] || TONES.neutral,
          )}
        >
          <Icon size={17} />
        </span>
      )}
      <span className="min-w-0 flex-1">
        <span
          className={cx(
            'block truncate text-sm font-semibold',
            danger ? 'text-danger-600' : 'text-ink',
          )}
        >
          {title}
        </span>
        {subtitle && <span className="mt-0.5 block truncate text-xs text-muted">{subtitle}</span>}
      </span>
      {value && <span className="shrink-0 text-xs font-medium text-muted">{value}</span>}
      {trailing}
      {showChevron && <ChevronRight size={17} className="shrink-0 text-faint" />}
    </Tag>
  )
}
