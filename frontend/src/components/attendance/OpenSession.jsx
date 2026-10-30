import { useEffect, useMemo, useState } from 'react'
import { Bluetooth, Clock, MapPin, Users } from 'lucide-react'
import { Alert, Button, Card, StatusBadge } from '../ui'
import cx from '../../lib/cx'
import { formatClock } from '../../lib/time'
import { DURATIONS, bluetoothState, positionState, sectionLabel } from '../../lib/attendance'

const minutesLeft = (endsAt, now) => {
  const end = new Date(endsAt).getTime()
  if (Number.isNaN(end)) return 0
  return Math.max(0, Math.round((end - now.getTime()) / 60000))
}

const TONES = {
  good: 'text-success-700',
  warn: 'text-warning-700',
  bad: 'text-danger-600',
  idle: 'text-muted',
}

const FIX_LABELS = {
  ask: 'Allow it',
  settings: 'Open app settings',
  location: 'Open location settings',
  bluetooth: 'Turn Bluetooth on',
}

function StateTile({ icon: Icon, label, state, onFix }) {
  return (
    <div className="rounded-tile bg-wash px-4 py-3">
      <p className="label-caps">{label}</p>
      <p
        className={cx(
          'mt-1 flex items-center gap-2 text-sm font-semibold',
          TONES[state.tone] || TONES.idle,
        )}
      >
        <Icon size={16} className="shrink-0" />
        <span className="wrap-long">{state.label}</span>
      </p>
      {state.reason && <p className="mt-1 text-xs text-muted">{state.reason}</p>}
      {state.fix && (
        <Button variant="ghost" size="sm" className="mt-1 -ml-2" onClick={() => onFix(state.fix)}>
          {FIX_LABELS[state.fix]}
        </Button>
      )}
    </div>
  )
}

export default function OpenSession({
  session,
  marks,
  now,
  status,
  busy,
  onExtend,
  onClose,
  onFix,
}) {
  const [extra, setExtra] = useState(10)
  const [showCode, setShowCode] = useState(false)
  const left = minutesLeft(session.ends_at, now)
  const present = session.present || 0
  const roster = session.roster || 0
  const latest = useMemo(
    () =>
      marks
        .filter((mark) => mark.status === 'present' && mark.marked_at)
        .sort((first, second) => new Date(second.marked_at) - new Date(first.marked_at))
        .slice(0, 8),
    [marks],
  )
  useEffect(() => {
    if (session.status !== 'open') setShowCode(false)
  }, [session.status])
  const bluetooth = bluetoothState(status)
  const position = positionState(session, status, now)
  return (
    <Card hover={false} rule className="animate-fade-in-up">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="label-caps">Attendance open</p>
          <p className="font-display mt-1 text-section font-bold text-ink">
            {session.course_code} · {session.course_title}
          </p>
          <p className="mt-1 text-xs text-muted">
            {session.batch} · {sectionLabel(session.section)} · started{' '}
            {formatClock(session.started_at)}
          </p>
        </div>
        <StatusBadge status="active" label={`${left} min left`} />
      </div>
      <div className="mt-4 grid gap-3 sm:grid-cols-3">
        <div className="rounded-tile bg-wash px-4 py-3">
          <p className="label-caps">Marked</p>
          <p className="font-display mt-1 text-title font-bold text-ink">
            {present}
            <span className="text-section text-muted">/{roster}</span>
          </p>
        </div>
        <StateTile icon={Bluetooth} label="Bluetooth" state={bluetooth} onFix={onFix} />
        <StateTile icon={MapPin} label="Position" state={position} onFix={onFix} />
      </div>
      {bluetooth.tone !== 'good' && (
        <Alert tone="warning" className="mt-4">
          Students near you can still be marked with the 4-digit code. Tap Show code and read it out
          or put it on the board.
        </Alert>
      )}
      {showCode ? (
        <div className="mt-4 rounded-card border border-brand-100 bg-brand-50 px-4 py-6 text-center">
          <p className="label-caps">Code on the board</p>
          <p className="font-display mt-2 text-[3.5rem] leading-none font-bold tracking-[0.3em] text-brand-700">
            {session.code || '····'}
          </p>
          <p className="mt-3 flex items-center justify-center gap-1.5 text-xs text-muted">
            <Clock size={13} /> New code in {session.code_seconds || 0} s
          </p>
        </div>
      ) : null}
      <div className="mt-4 flex flex-col gap-3">
        <Button
          variant="secondary"
          size="sm"
          className="self-start"
          onClick={() => setShowCode((open) => !open)}
        >
          {showCode ? 'Hide code' : 'Show code'}
        </Button>
        <div>
          <p className="label-caps">Keep it open longer</p>
          <div
            role="radiogroup"
            aria-label="Extra minutes"
            className="no-scrollbar -mx-4 mt-2 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:px-0"
          >
            {DURATIONS.map((value) => {
              const active = value === extra
              return (
                <button
                  key={value}
                  type="button"
                  role="radio"
                  aria-checked={active}
                  onClick={() => setExtra(value)}
                  className={cx(
                    'press min-h-10 shrink-0 rounded-full border px-4 text-sm whitespace-nowrap',
                    active
                      ? 'border-brand-600 bg-brand-600 font-semibold text-page'
                      : 'glass-chip font-medium text-muted hover:border-brand-300 hover:text-ink',
                  )}
                >
                  {value}m
                </button>
              )
            })}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" size="sm" loading={busy === 'extend'} onClick={() => onExtend(extra)}>
            Extend {extra}m
          </Button>
          <Button variant="danger" size="sm" loading={busy === 'close'} onClick={onClose}>
            Close attendance
          </Button>
        </div>
      </div>
      <div className="mt-5">
        <p className="label-caps flex items-center gap-1.5">
          <Users size={13} /> Latest marks
        </p>
        {latest.length === 0 ? (
          <p className="mt-2 text-sm text-muted">Nobody has been marked yet.</p>
        ) : (
          <ul className="mt-2 flex flex-col gap-1.5">
            {latest.map((mark) => (
              <li
                key={mark.student_id}
                className="flex items-center justify-between gap-3 rounded-tile bg-wash px-3 py-2 text-sm"
              >
                <span className="min-w-0 truncate font-medium text-ink">
                  {mark.full_name}
                  <span className="ml-2 text-xs text-muted">{mark.reg_no}</span>
                </span>
                <span className="shrink-0 text-xs text-muted">
                  {formatClock(mark.marked_at)}
                  {mark.distance_m ? ` · ${mark.distance_m} m` : ''}
                  {mark.method === 'code' ? ' · code' : ''}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}
