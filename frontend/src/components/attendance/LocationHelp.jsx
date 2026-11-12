import { MapPin } from 'lucide-react'
import { Button } from '../ui'
import cx from '../../lib/cx'

const TONES = {
  good: 'text-success-700',
  warn: 'text-warning-700',
  bad: 'text-danger-600',
  idle: 'text-muted',
}

const FIX_LABELS = {
  allow: 'Allow location',
  retry: 'Try again',
  ask: 'Allow it',
  settings: 'Open app settings',
  location: 'Open location settings',
  bluetooth: 'Turn Bluetooth on',
}

export default function LocationHelp({ state, label, icon: Icon = MapPin, onFix, busy, className }) {
  if (!state) return null
  return (
    <div className={cx('rounded-tile bg-wash px-4 py-3', className)}>
      {label && <p className="label-caps">{label}</p>}
      <p
        className={cx(
          'mt-1 flex items-center gap-2 text-sm font-semibold',
          TONES[state.tone] || TONES.idle,
        )}
        role={state.tone === 'bad' ? 'status' : undefined}
      >
        <Icon size={16} className="shrink-0" />
        <span className="wrap-long">{state.label}</span>
      </p>
      {state.reason && <p className="mt-1 text-xs text-ink-soft">{state.reason}</p>}
      {state.steps?.length > 0 && (
        <ol className="mt-1.5 list-decimal space-y-1 pl-5 text-xs text-ink-soft">
          {state.steps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
      )}
      {state.fix && onFix && (
        <Button
          variant={state.fix === 'allow' ? 'primary' : 'secondary'}
          size="sm"
          className="mt-2"
          loading={busy}
          onClick={() => onFix(state.fix)}
        >
          {FIX_LABELS[state.fix]}
        </Button>
      )}
    </div>
  )
}
