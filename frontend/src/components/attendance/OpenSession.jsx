import { QUICK_EXTENSIONS, extensionProblem, lateBoundaryNote } from '../../lib/attendanceTiming'
import { useEffect, useMemo, useState } from 'react'
import { Bluetooth, Clock, MapPin, Users } from 'lucide-react'
import { Alert, Button, Card, StatusBadge } from '../ui'
import LocationHelp from './LocationHelp'
import cx from '../../lib/cx'
import { dots } from '../../lib/dots'
import { formatClock } from '../../lib/time'
import {
  CODE_ONLY_LABEL,
  DURATIONS,
  bluetoothState,
  markFlags,
  markMeta,
  methodLabel,
  positionState,
  sectionLabel,
} from '../../lib/attendance'
import { WEB_BLUETOOTH_LINE, browserEnv, webPositionState } from '../../lib/webLocation'

const minutesLeft = (endsAt, now) => {
  const end = new Date(endsAt).getTime()
  if (Number.isNaN(end)) return 0
  return Math.max(0, Math.round((end - now.getTime()) / 60000))
}

export default function OpenSession({
  session,
  marks,
  now,
  status,
  place,
  app,
  busy,
  onExtend,
  onClose,
  onFix,
  onCodeOnly,
}) {
  const [extra, setExtra] = useState(10)
  const [custom, setCustom] = useState('')
  const [customOpen, setCustomOpen] = useState(false)
  const extraMinutes = customOpen ? Number(custom) : extra
  const extraProblem = customOpen ? extensionProblem(custom) : ''
  const [showCode, setShowCode] = useState(!app)
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
  const position = app
    ? positionState(session, status, now)
    : webPositionState(place, session, now, browserEnv())
  const offerCodeOnly = !session.code_only && position?.tone === 'bad' && Boolean(onCodeOnly)
  return (
    <Card hover={false} rule className="animate-fade-in-up">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="label-caps">Attendance open</p>
          <p className="font-display mt-1 text-section font-bold text-ink">
            {dots(session.course_code, session.course_title)}
          </p>
          <p className="mt-1 text-xs text-muted">
            {dots(
              session.batch,
              sectionLabel(session.section),
              session.started_at ? `started ${formatClock(session.started_at)}` : '',
            )}
          </p>
        </div>
        <StatusBadge status="active" label={`${left} min left`} />
      </div>
      <div className={cx('mt-4 grid gap-3', app ? 'sm:grid-cols-3' : 'sm:grid-cols-2')}>
        <div className="rounded-tile bg-wash px-4 py-3">
          <p className="label-caps">Marked</p>
          <p className="font-display mt-1 text-title font-bold text-ink">
            {present}
            <span className="text-section text-muted">/{roster}</span>
          </p>
        </div>
        {app && !session.code_only && (
          <LocationHelp icon={Bluetooth} label="Bluetooth" state={bluetooth} onFix={onFix} />
        )}
        <LocationHelp
          icon={MapPin}
          label="Position"
          state={position}
          onFix={onFix}
          busy={busy === 'locate'}
        />
      </div>
      {!app && <p className="mt-3 text-xs text-muted">{WEB_BLUETOOTH_LINE}</p>}
      {app && !session.code_only && bluetooth.tone !== 'good' && (
        <Alert tone="warning" className="mt-4">
          Students near you can still be marked with the 4-digit code. Tap Show code and read it out
          or put it on the board.
        </Alert>
      )}
      {session.code_only && (
        <Alert tone="info" className="mt-4">
          {CODE_ONLY_LABEL}. Students mark themselves with the code alone, and every mark is flagged
          No location in the list and the Excel sheet.
        </Alert>
      )}
      {offerCodeOnly && (
        <div className="mt-3">
          <Button
            variant="secondary"
            size="sm"
            loading={busy === 'code-only'}
            onClick={onCodeOnly}
          >
            Switch to {CODE_ONLY_LABEL.toLowerCase()}
          </Button>
        </div>
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
          variant={app ? 'secondary' : 'primary'}
          size={app ? 'sm' : 'md'}
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
            {QUICK_EXTENSIONS.map((value) => {
              const active = !customOpen && value === extra
              return (
                <button
                  key={value}
                  type="button"
                  role="radio"
                  aria-checked={active}
                  onClick={() => {
                    setCustomOpen(false)
                    setExtra(value)
                  }}
                  className={cx(
                    'press min-h-10 shrink-0 rounded-full border px-4 text-sm whitespace-nowrap',
                    active
                      ? 'border-brand-600 bg-brand-600 font-semibold text-page'
                      : 'glass-chip font-medium text-muted hover:border-brand-300 hover:text-ink',
                  )}
                >
                  +{value} min
                </button>
              )
            })}
            <button
              type="button"
              role="radio"
              aria-checked={customOpen}
              onClick={() => setCustomOpen(true)}
              className={cx(
                'press min-h-10 shrink-0 rounded-full border px-4 text-sm whitespace-nowrap',
                customOpen
                  ? 'border-brand-600 bg-brand-600 font-semibold text-page'
                  : 'glass-chip font-medium text-muted hover:border-brand-300 hover:text-ink',
              )}
            >
              Custom
            </button>
          </div>
          {customOpen && (
            <label className="mt-3 flex flex-wrap items-center gap-2 text-sm text-ink">
              <input
                type="number"
                inputMode="numeric"
                min={1}
                max={240}
                value={custom}
                onChange={(event) => setCustom(event.target.value)}
                aria-label="Extra minutes to add"
                placeholder="7"
                className="h-11 w-28 rounded-xl border border-line-strong bg-wash px-3 text-sm text-ink tabular-nums outline-none focus:border-brand-600"
              />
              more minutes
            </label>
          )}
          {lateBoundaryNote(session) && (
            <p className="mt-2 text-xs text-muted">{lateBoundaryNote(session)}</p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="secondary"
            size="sm"
            loading={busy === 'extend'}
            disabled={Boolean(extraProblem)}
            onClick={() => onExtend(extraMinutes)}
          >
            Extend {extraProblem ? '' : `${extraMinutes} min`}
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
                <span className="min-w-0 wrap-long font-medium text-ink">
                  {mark.full_name}
                  <span className="ml-2 text-xs text-muted">{mark.reg_no}</span>
                </span>
                <span className="shrink-0 text-xs text-muted">
                  {dots(markMeta(mark, formatClock), methodLabel(mark.method, session?.method))}
                  {markFlags(mark).map((flag) => (
                    <span key={flag} className="ml-1 font-semibold text-warning-700">
                      · {flag}
                    </span>
                  ))}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}
