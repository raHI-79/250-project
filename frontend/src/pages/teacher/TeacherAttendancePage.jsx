import { useCallback, useEffect, useRef, useState } from 'react'
import { CalendarCheck, Download, Radio } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  EmptyState,
  Field,
  PageHeader,
  Segmented,
  Select,
  SkeletonCard,
  StatusTabs,
  useToast,
} from '../../components/ui'
import OpenSession from '../../components/attendance/OpenSession'
import SessionSheet from '../../components/attendance/SessionSheet'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { useNow, usePolling, useResource } from '../../lib/hooks'
import { formatDate, formatClock } from '../../lib/time'
import {
  DEFAULT_MINUTES,
  DURATIONS,
  TEACHER_TAB_CLASSES,
  TEACHER_TAB_HISTORY,
  TEACHER_TAB_TAKE,
  askBeacon,
  askBeaconLocation,
  beaconStatus,
  beaconSupported,
  belowPass,
  classKey,
  downloadWorkbook,
  openAppSettings,
  openLocationSettings,
  sectionLabel,
  startBeacon,
  stopBeacon,
  turnBluetoothOn,
  updateBeacon,
} from '../../lib/attendance'

const TABS = [
  { value: TEACHER_TAB_TAKE, label: 'Take attendance' },
  { value: TEACHER_TAB_HISTORY, label: 'History' },
  { value: TEACHER_TAB_CLASSES, label: 'My classes' },
]

const OFF_BROWSER = 'The website shows the code for a projector. Bluetooth needs the Android app.'

const WEB_STATUS = {
  supported: false,
  bluetooth_on: false,
  nearby: false,
  place: false,
  location_on: false,
  advertising: false,
  asked: true,
  error: OFF_BROWSER,
}

function StartPanel({ options, minutes, setMinutes, picked, setPicked, busy, onStart }) {
  const today = options?.today || []
  const classes = options?.classes || []
  const choices = today.length > 0 ? today : classes
  return (
    <Card hover={false}>
      <CardHeader title="Start attendance" className="px-0 pt-0" />
      {choices.length === 0 ? (
        <p className="text-sm text-muted">
          You have no class in the running semester, so there is nothing to take attendance for.
        </p>
      ) : (
        <div className="flex flex-col gap-4">
          {today.length > 0 && (
            <div>
              <p className="label-caps">Today</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {today.map((entry) => {
                  const key = classKey(entry)
                  const active = picked === key
                  return (
                    <button
                      key={`${entry.schedule_item_id}-${key}`}
                      type="button"
                      onClick={() => setPicked(key)}
                      className={cx(
                        'press rounded-tile border px-3 py-2 text-left text-sm transition-colors',
                        active
                          ? 'border-brand-600 bg-brand-50 font-semibold text-ink'
                          : 'border-line-strong bg-surface text-ink hover:bg-wash',
                      )}
                    >
                      <span className="block font-semibold">{entry.course_code}</span>
                      <span className="mt-0.5 block text-xs text-muted">
                        {entry.batch} · {sectionLabel(entry.section)}
                        {entry.start_time ? ` · ${entry.start_time}` : ''}
                      </span>
                    </button>
                  )
                })}
              </div>
            </div>
          )}
          <Field label="Class" htmlFor="attendance-class">
            <Select
              id="attendance-class"
              value={picked}
              onChange={(event) => setPicked(event.target.value)}
            >
              <option value="">Pick a class</option>
              {classes.map((entry) => (
                <option key={classKey(entry)} value={classKey(entry)}>
                  {entry.course_code} · {entry.batch} · {sectionLabel(entry.section)}
                </option>
              ))}
            </Select>
          </Field>
          <div>
            <p className="label-caps">How long it stays open</p>
            <Segmented
              className="mt-2 max-w-md"
              label="Duration"
              options={DURATIONS.map((value) => ({ value, label: `${value} min` }))}
              value={minutes}
              onChange={setMinutes}
            />
          </div>
          <div>
            <Button loading={busy} disabled={!picked} onClick={onStart}>
              <Radio size={16} /> Start attendance
            </Button>
          </div>
        </div>
      )}
    </Card>
  )
}

function HistoryList({ sessions, loading, onOpen }) {
  if (loading) return <SkeletonCard />
  if (sessions.length === 0) {
    return (
      <EmptyState
        icon={CalendarCheck}
        title="No attendance yet"
        text="Start attendance in a class and it shows up here with every student, the time and the distance."
      />
    )
  }
  return (
    <Card hover={false} flush>
      <ul className="divide-y divide-line">
        {sessions.map((session) => (
          <li key={session.id}>
            <button
              type="button"
              onClick={() => onOpen(session.id)}
              className="press flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-wash"
            >
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-semibold text-ink">
                  {session.course_code} · {session.course_title}
                </span>
                <span className="mt-0.5 block truncate text-xs text-muted">
                  {session.batch} · {sectionLabel(session.section)} ·{' '}
                  {formatDate(session.started_at)} · {formatClock(session.started_at)}
                </span>
              </span>
              <span className="shrink-0 text-sm font-semibold text-ink">
                {session.present}/{session.roster}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </Card>
  )
}

function ClassList({ classes, loading, onExport, busy }) {
  if (loading) return <SkeletonCard />
  if (classes.length === 0) {
    return (
      <EmptyState
        icon={CalendarCheck}
        title="No classes yet"
        text="Once you take attendance, every course, batch and section shows its average here."
      />
    )
  }
  return (
    <Card hover={false} flush>
      <ul className="divide-y divide-line">
        {classes.map((entry) => (
          <li key={classKey(entry)} className="flex items-center gap-3 px-4 py-3">
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-semibold text-ink">
                {entry.course_code} · {entry.course_title}
              </span>
              <span className="mt-0.5 block truncate text-xs text-muted">
                {entry.batch} · {sectionLabel(entry.section)} · {entry.sessions} session
                {entry.sessions === 1 ? '' : 's'} · {entry.below_fifty} below 50 %
              </span>
            </span>
            <span
              className={cx(
                'shrink-0 text-sm font-bold',
                belowPass(entry.average_rate) ? 'text-danger-600' : 'text-ink',
              )}
            >
              {entry.average_rate} %
            </span>
            <Button
              variant="secondary"
              size="sm"
              loading={busy === classKey(entry)}
              onClick={() => onExport(entry)}
            >
              <Download size={15} /> Excel
            </Button>
          </li>
        ))}
      </ul>
    </Card>
  )
}

export default function TeacherAttendancePage() {
  const toast = useToast()
  const now = useNow(5000)
  const [tab, setTab] = useState(TEACHER_TAB_TAKE)
  const [minutes, setMinutes] = useState(DEFAULT_MINUTES)
  const [picked, setPicked] = useState('')
  const [busy, setBusy] = useState('')
  const [open, setOpen] = useState(null)
  const [marks, setMarks] = useState([])
  const [detailID, setDetailID] = useState(0)
  const [status, setStatus] = useState(null)
  const broadcasting = useRef(0)
  const asked = useRef(false)
  const options = useResource('/api/teacher/attendance/options')
  const history = useResource('/api/teacher/attendance/sessions')
  const classes = useResource('/api/teacher/attendance/classes')
  useEffect(() => {
    if (options.data?.open) setOpen(options.data.open)
  }, [options.data])
  const refreshOpen = useCallback(async () => {
    if (!open?.id) return
    try {
      const data = await api.get(`/api/teacher/attendance/sessions/${open.id}`)
      setMarks(data?.marks || [])
      if (data?.session?.status === 'open') {
        setOpen(data.session)
        if (broadcasting.current === data.session.id) {
          updateBeacon(data.session).catch(() => null)
        }
      } else {
        setOpen(null)
        setMarks([])
      }
    } catch {
      return
    }
  }, [open?.id])
  usePolling(open?.id ? refreshOpen : null, 5000)
  const readStatus = useCallback(async () => {
    if (!beaconSupported()) {
      setStatus(WEB_STATUS)
      return WEB_STATUS
    }
    try {
      const fresh = await beaconStatus()
      const seen = { ...fresh, asked: asked.current }
      setStatus(seen)
      return seen
    } catch {
      return null
    }
  }, [])
  usePolling(beaconSupported() && open?.id ? readStatus : null, 4000)
  const prepare = useCallback(
    async (session) => {
      if (!beaconSupported()) {
        setStatus(WEB_STATUS)
        return
      }
      try {
        const place = await askBeaconLocation()
        asked.current = true
        setStatus({ ...place, asked: true })
      } catch {
        asked.current = true
      }
      try {
        const nearby = await askBeacon()
        asked.current = true
        if (nearby?.supported && nearby?.bluetooth_on && nearby?.nearby && session?.token_key) {
          await startBeacon(session)
          broadcasting.current = session.id
        }
      } catch (failure) {
        toast.error(failure.message)
      }
      readStatus()
    },
    [readStatus, toast],
  )
  const fix = async (what) => {
    try {
      if (what === 'ask') await prepare(open)
      else if (what === 'bluetooth') await turnBluetoothOn()
      else if (what === 'settings') await openAppSettings()
      else if (what === 'location') await openLocationSettings()
      await readStatus()
      if (what === 'bluetooth' && open?.token_key) await prepare(open)
    } catch (failure) {
      toast.error(failure.message)
    }
  }
  useEffect(() => {
    if (!open?.id) return
    if (!open.token_key) {
      toast.error('This session did not come with its Bluetooth key. Close it and start again.')
      return
    }
    if (broadcasting.current === open.id) return
    prepare(open)
  }, [open, prepare, toast])
  useEffect(
    () => () => {
      if (broadcasting.current && beaconSupported()) stopBeacon().catch(() => null)
    },
    [],
  )
  const start = async () => {
    const entry = (options.data?.today || []).concat(options.data?.classes || [])
      .find((item) => classKey(item) === picked)
    if (!entry) return
    setBusy('start')
    try {
      const session = await api.post('/api/teacher/attendance/sessions', {
        course_id: entry.course_id,
        batch: entry.batch,
        section: entry.section,
        schedule_item_id: entry.schedule_item_id || 0,
        minutes,
      })
      setOpen(session)
      setMarks([])
      toast.success('Attendance is open.')
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const extend = async (extra) => {
    setBusy('extend')
    try {
      const session = await api.post(
        `/api/teacher/attendance/sessions/${open.id}/extend`,
        { minutes: extra },
      )
      setOpen(session)
      toast.success('Attendance stays open longer.')
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const close = async () => {
    setBusy('close')
    try {
      await api.post(`/api/teacher/attendance/sessions/${open.id}/close`)
      if (beaconSupported()) await stopBeacon().catch(() => null)
      broadcasting.current = 0
      setStatus(null)
      setOpen(null)
      setMarks([])
      toast.success('Attendance closed. Everyone missing is marked absent.')
      history.reload()
      classes.reload()
      options.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const exportOne = async (entry) => {
    const key = entry ? classKey(entry) : 'all'
    setBusy(key)
    try {
      const saved = await downloadWorkbook(
        '/api/teacher/attendance/export',
        entry
          ? { course_id: entry.course_id, batch: entry.batch, section: entry.section }
          : {},
        entry ? `attendance_${entry.course_code}_${entry.batch}-${entry.section}.xlsx` : 'attendance_all-classes.xlsx',
      )
      toast.success(saved.path ? `Saved to Documents as ${saved.name}` : 'The workbook is downloading.')
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="Your classes"
        title="Attendance"
        subtitle="Open attendance for a class, let the students mark themselves from their own phone, and download the sheet."
        actions={
          <Button
            variant="secondary"
            loading={busy === 'all'}
            onClick={() => exportOne(null)}
          >
            <Download size={16} /> Download all my classes
          </Button>
        }
      />
      <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-5" />
      {options.error && (
        <Alert tone="error" className="mb-5">
          {options.error}
        </Alert>
      )}
      {tab === TEACHER_TAB_TAKE && (
        <div className="flex flex-col gap-5">
          {open ? (
            <OpenSession
              session={open}
              marks={marks}
              now={now}
              status={status}
              busy={busy}
              onExtend={extend}
              onClose={close}
              onFix={fix}
            />
          ) : options.loading ? (
            <SkeletonCard />
          ) : (
            <StartPanel
              options={options.data}
              minutes={minutes}
              setMinutes={setMinutes}
              picked={picked}
              setPicked={setPicked}
              busy={busy === 'start'}
              onStart={start}
            />
          )}
        </div>
      )}
      {tab === TEACHER_TAB_HISTORY && (
        <HistoryList
          sessions={history.data?.items || []}
          loading={history.loading}
          onOpen={setDetailID}
        />
      )}
      {tab === TEACHER_TAB_CLASSES && (
        <ClassList
          classes={classes.data?.items || []}
          loading={classes.loading}
          busy={busy}
          onExport={exportOne}
        />
      )}
      <SessionSheet
        sessionId={detailID}
        onClose={() => setDetailID(0)}
        onChanged={() => {
          history.reload()
          classes.reload()
        }}
      />
    </>
  )
}
