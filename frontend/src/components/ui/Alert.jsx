import { AlertTriangle, CheckCircle2, Info, XCircle } from 'lucide-react'
import cx from '../../lib/cx'

const tones = {
  success: {
    icon: CheckCircle2,
    className: 'border-l-[3px] border-emerald-600 bg-emerald-50 text-emerald-800',
    iconClass: 'text-emerald-600',
  },
  error: {
    icon: XCircle,
    className: 'border-l-[3px] border-accent-600 bg-accent-50 text-accent-700',
    iconClass: 'text-accent-600',
  },
  info: {
    icon: Info,
    className: 'border-l-[3px] border-brand-600 bg-brand-50 text-ink-soft',
    iconClass: 'text-brand-600',
  },
  warning: {
    icon: AlertTriangle,
    className: 'border-l-[3px] border-gold-500 bg-gold-100 text-gold-700',
    iconClass: 'text-gold-600',
  },
}

export default function Alert({ tone = 'info', title, children, className }) {
  const config = tones[tone] || tones.info
  const Icon = config.icon
  return (
    <div
      role={tone === 'error' ? 'alert' : 'status'}
      className={cx(
        'animate-fade-in flex items-start gap-3 border border-line px-4 py-3 text-sm',
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
