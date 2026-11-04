import cx from '../../lib/cx'

const TONES = {
  brand: 'bg-brand-50 text-brand-700',
  success: 'bg-success-50 text-success-600',
  warning: 'bg-warning-50 text-warning-600',
  danger: 'bg-danger-50 text-danger-600',
  purple: 'bg-purple-50 text-purple-600',
  neutral: 'bg-wash text-ink-soft',
  muted: 'bg-wash text-muted',
}

const styles = {
  pending: { label: 'Pending', className: TONES.warning },
  approved: { label: 'Approved', className: TONES.success },
  rejected: { label: 'Rejected', className: TONES.danger },
  cancelled: { label: 'Cancelled', className: TONES.danger },
  auto_cancelled: { label: 'Auto cancelled', className: TONES.muted },
  active: { label: 'Active', className: TONES.brand },
  removed: { label: 'Removed', className: TONES.danger },
  yours: { label: 'Yours', className: TONES.purple },
  gold: { label: '', className: TONES.warning },
  unavailable: { label: 'Unavailable', className: TONES.neutral },
}

export default function StatusBadge({ status, label, tone, className }) {
  const config = styles[status] || { label: status, className: TONES.neutral }
  return (
    <span
      className={cx(
        'inline-flex items-center rounded-full px-2.5 py-1 text-2xs font-extrabold tracking-[0.04em] whitespace-nowrap uppercase',
        (tone && TONES[tone]) || config.className,
        className,
      )}
    >
      {label || config.label}
    </span>
  )
}
