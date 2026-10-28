import { Link } from 'react-router-dom'
import cx from '../../lib/cx'

const sizes = {
  sm: 'size-10 [&_svg]:size-4.5',
  md: 'size-12 [&_svg]:size-5',
}

const tones = {
  glass: 'glass-chip text-ink-soft hover:text-brand-700',
  plain: 'text-ink-soft hover:bg-wash hover:text-ink',
  brand: 'bg-brand-50 text-brand-600 hover:bg-brand-100',
  danger: 'text-ink-soft hover:bg-danger-50 hover:text-danger-600',
}

export default function IconButton({
  as,
  label,
  tone = 'plain',
  size = 'md',
  to,
  className,
  children,
  ...rest
}) {
  const Tag = as || (to ? Link : 'button')
  return (
    <Tag
      to={to}
      type={Tag === 'button' ? 'button' : undefined}
      aria-label={label}
      title={label}
      className={cx(
        'press grid shrink-0 place-items-center rounded-full',
        sizes[size] || sizes.md,
        tones[tone] || tones.plain,
        className,
      )}
      {...rest}
    >
      {children}
    </Tag>
  )
}
