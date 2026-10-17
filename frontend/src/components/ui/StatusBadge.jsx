import cx from '../../lib/cx'

const styles = {
  pending: { label: 'Pending', className: 'border-amber-200 bg-amber-50 text-amber-800' },
  approved: { label: 'Approved', className: 'border-brand-200 bg-brand-50 text-brand-700' },
  rejected: { label: 'Rejected', className: 'border-accent-200 bg-accent-50 text-accent-700' },
  cancelled: { label: 'Cancelled', className: 'border-brand-100 bg-brand-50 text-brand-500' },
  auto_cancelled: {
    label: 'Auto cancelled',
    className: 'border-brand-100 bg-brand-50 text-brand-500',
  },
  active: { label: 'Active', className: 'border-brand-200 bg-brand-50 text-brand-700' },
  removed: { label: 'Removed', className: 'border-accent-200 bg-accent-50 text-accent-700' },
  yours: { label: 'Yours', className: 'border-accent-200 bg-accent-50 text-accent-700' },
  unavailable: { label: 'Unavailable', className: 'border-brand-100 bg-brand-50 text-brand-400' },
}

export default function StatusBadge({ status, label, className }) {
  const config = styles[status] || { label: status, className: styles.unavailable.className }
  return (
    <span
      className={cx(
        'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium whitespace-nowrap',
        config.className,
        className,
      )}
    >
      {label || config.label}
    </span>
  )
}
