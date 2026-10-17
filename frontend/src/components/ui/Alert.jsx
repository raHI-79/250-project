import { AlertTriangle, CheckCircle2, Info, XCircle } from 'lucide-react'
import cx from '../../lib/cx'

const tones = {
  success: {
    icon: CheckCircle2,
    className: 'border-brand-200 bg-brand-50 text-brand-800',
    iconClass: 'text-brand-600',
  },
  error: {
    icon: XCircle,
    className: 'border-accent-200 bg-accent-50 text-accent-800',
    iconClass: 'text-accent-600',
  },
  info: {
    icon: Info,
    className: 'border-brand-100 bg-white text-brand-800',
    iconClass: 'text-brand-500',
  },
  warning: {
    icon: AlertTriangle,
    className: 'border-amber-200 bg-amber-50 text-amber-900',
    iconClass: 'text-amber-600',
  },
}

export default function Alert({ tone = 'info', title, children, className }) {
  const config = tones[tone] || tones.info
  const Icon = config.icon
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={cx(
        'animate-fade-in flex items-start gap-3 rounded-lg border px-4 py-3 text-sm',
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
