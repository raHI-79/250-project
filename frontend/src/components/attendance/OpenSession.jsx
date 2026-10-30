import { useEffect, useMemo, useState } from 'react'
import { Bluetooth, Clock, MapPin, Users } from 'lucide-react'
import { Alert, Button, Card, Segmented, StatusBadge } from '../ui'
import cx from '../../lib/cx'
import { formatClock } from '../../lib/time'
import { DURATIONS, sectionLabel } from '../../lib/attendance'

const minutesLeft = (endsAt, now) => {
  const end = new Date(endsAt).getTime()
  if (Number.isNaN(end)) return 0
  return Math.max(0, Math.round((end - now.getTime()) / 60000))
}

export default function OpenSession({
  session,
  marks,
  now,
  beacon,
  busy,
  onExtend,
  onClose,
  onFixBluetooth,
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
  const stale =
    !session.location_updated_at ||
    now.getTime() - new Date(session.location_updated_at).getTime() > 60000
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
        <div className="rounded-tile bg-wash px-4 py-3">
          <p className="label-caps">Bluetooth</p>
          <p
            className={cx(
              'mt-1 flex items-center gap-2 text-sm font-semibold',
              beacon.running ? 'text-success-700' : 'text-muted',
            )}
          >
            <Bluetooth size={16} />
            {beacon.running ? 'Broadcasting' : beacon.reason || 'Off'}
          </p>
        </div>
        <div className="rounded-tile bg-wash px-4 py-3">
          <p className="label-caps">Position</p>
          <p
            className={cx(
              'mt-1 flex items-center gap-2 text-sm font-semibold',
              stale ? 'text-warning-700' : 'text-success-700',
            )}
          >
            <MapPin size={16} />
            {session.has_location ? (stale ? 'Out of date' : 'Fresh') : 'Waiting'}
          </p>
        </div>
      </div>
      {!beacon.running && beacon.reason && (
        <Alert tone="warning" className="mt-4">
          {beacon.reason}
          {beacon.fixable && (
            <Button
              variant="ghost"
              size="sm"
              className="mt-2"
              onClick={onFixBluetooth}
            >
              Turn Bluetooth on
            </Button>
          )}
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
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Button variant="secondary" size="sm" onClick={() => setShowCode((open) => !open)}>
          {showCode ? 'Hide code' : 'Show code'}
        </Button>
        <div className="flex items-center gap-2">
          <Segmented
            label="Extra minutes"
            options={DURATIONS.map((value) => ({ value, label: `${value}m` }))}
            value={extra}
            onChange={setExtra}
            className="w-56"
          />
          <Button variant="secondary" size="sm" loading={busy === 'extend'} onClick={() => onExtend(extra)}>
            Extend
          </Button>
        </div>
        <Button variant="danger" size="sm" loading={busy === 'close'} onClick={onClose}>
          Close attendance
        </Button>
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
