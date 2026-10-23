import cx from '../../lib/cx'

const styles = {
  pending: { label: 'Pending', className: 'bg-warning-50 text-warning-700' },
  approved: { label: 'Approved', className: 'bg-success-50 text-success-700' },
  rejected: { label: 'Rejected', className: 'bg-danger-50 text-danger-700' },
  cancelled: { label: 'Cancelled', className: 'bg-danger-50 text-danger-700' },
  auto_cancelled: { label: 'Auto cancelled', className: 'bg-wash text-muted' },
  active: { label: 'Active', className: 'bg-brand-50 text-brand-700' },
  removed: { label: 'Removed', className: 'bg-danger-50 text-danger-700' },
  yours: { label: 'Yours', className: 'bg-gold-50 text-gold-700' },
  gold: { label: '', className: 'bg-gold-50 text-gold-700' },
  unavailable: { label: 'Unavailable', className: 'bg-wash text-ink-soft' },
}

export default function StatusBadge({ status, label, className }) {
  const config = styles[status] || { label: status, className: styles.unavailable.className }
  return (
    <span
      className={cx(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-2xs font-bold tracking-[0.04em] whitespace-nowrap uppercase',
        config.className,
        className,
      )}
    >
      {label || config.label}
    </span>
  )
}
