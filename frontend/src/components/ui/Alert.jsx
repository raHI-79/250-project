import { AlertTriangle, CheckCircle2, Info, XCircle } from 'lucide-react'
import cx from '../../lib/cx'

const tones = {
  success: {
    icon: CheckCircle2,
    className: 'border-success-100 bg-success-50 text-success-700',
    iconClass: 'text-success-600',
  },
  error: {
    icon: XCircle,
    className: 'border-danger-100 bg-danger-50 text-danger-700',
    iconClass: 'text-danger-600',
  },
  info: {
    icon: Info,
    className: 'border-brand-100 bg-brand-50 text-ink-soft',
    iconClass: 'text-brand-600',
  },
  warning: {
    icon: AlertTriangle,
    className: 'border-warning-100 bg-warning-50 text-warning-700',
    iconClass: 'text-warning-600',
  },
}

export default function Alert({ tone = 'info', title, children, className }) {
  const config = tones[tone] || tones.info
  const Icon = config.icon
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={cx(
        'animate-fade-in flex items-start gap-3 rounded-card border px-4 py-3 text-sm',
        config.className,
        className,
      )}
    >
      <Icon size={18} className={cx('mt-0.5 shrink-0', config.iconClass)} />
      <div className="min-w-0">
        {title && <p className="font-semibold">{title}</p>}
        {children && <div className={cx(title && 'mt-0.5')}>{children}</div>}
      </div>
    </div>
  )
}
