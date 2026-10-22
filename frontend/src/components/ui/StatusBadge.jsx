import cx from '../../lib/cx'

const styles = {
  pending: { label: 'Pending', className: 'bg-amber-100 text-amber-800' },
  approved: { label: 'Approved', className: 'bg-emerald-100 text-emerald-700' },
  rejected: { label: 'Rejected', className: 'bg-accent-100 text-accent-600' },
  cancelled: { label: 'Cancelled', className: 'bg-accent-100 text-accent-600' },
  auto_cancelled: { label: 'Auto cancelled', className: 'bg-wash text-muted' },
  active: { label: 'Active', className: 'bg-brand-100 text-brand-900' },
  removed: { label: 'Removed', className: 'bg-accent-100 text-accent-600' },
  yours: { label: 'Yours', className: 'bg-gold-100 text-gold-700' },
  gold: { label: '', className: 'bg-gold-100 text-gold-700' },
  unavailable: { label: 'Unavailable', className: 'bg-[#EEF2F7] text-slate-700' },
}

export default function StatusBadge({ status, label, className }) {
  const config = styles[status] || { label: status, className: styles.unavailable.className }
  return (
    <span
      className={cx(
        'inline-flex items-center rounded-[3px] px-2 py-0.5 text-xs font-semibold tracking-[0.04em] whitespace-nowrap uppercase',
        config.className,
        className,
      )}
    >
      {label || config.label}
    </span>
  )
}
