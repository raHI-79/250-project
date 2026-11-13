import { useCallback, useEffect, useRef, useState } from 'react'
import { CalendarCheck, Download, Radio, Trash2 } from 'lucide-react'
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
  useConfirm,
  useToast,
} from '../../components/ui'
import OpenSession from '../../components/attendance/OpenSession'
import RollCall from '../../components/attendance/RollCall'
import SessionSheet from '../../components/attendance/SessionSheet'
import PhoneRequestList from '../../components/attendance/PhoneRequestList'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { useNow, usePolling, useQueryTab, useResource } from '../../lib/hooks'
import { formatDate, formatClock } from '../../lib/time'
import useWebLocation from '../../lib/useWebLocation'
import { dots } from '../../lib/dots'
import {
  CODE_ONLY_LABEL,
  DEFAULT_MINUTES,
  TAKE_BLUETOOTH,
  TAKE_ROLL_CALL,
  TEACHER_TAB_CLASSES,
  TEACHER_TAB_HISTORY,
  TEACHER_TAB_PHONES,
  TEACHER_TAB_TAKE,
  deleteSheetQuestion,
  askBeacon,
  askBeaconLocation,
  beaconStatus,
  beaconSupported,
  belowPass,
  classKey,
  closedMessage,
  downloadWorkbook,
  openAppSettings,
  openLocationSettings,
  phoneId,
  presentOfRoster,
  sectionLabel,
  startBeacon,
  startBody,
  stopBeacon,
  takeMethods,
  turnBluetoothOn,
  updateBeacon,
} from '../../lib/attendance'
import SessionTiming from '../../components/attendance/SessionTiming'
import { hoursForClass, newTiming, timingBody, timingProblem } from '../../lib/attendanceTiming'

const TABS = [
  { value: TEACHER_TAB_TAKE, label: 'Take attendance' },
  { value: TEACHER_TAB_HISTORY, label: 'History' },
  { value: TEACHER_TAB_CLASSES, label: 'Sheets' },
  { value: TEACHER_TAB_PHONES, label: 'Phone requests' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

function StartPanel({
  options,
  timing,
  setTiming,
  problem,
  picked,
  setPicked,
  method,
  setMethod,
  codeOnly,
  setCodeOnly,
  app,
  busy,
  onStart,
}) {
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
                        {dots(entry.batch, sectionLabel(entry.section), entry.start_time)}
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
            <p className="label-caps">How attendance is taken</p>
            <Segmented
              className="mt-2 max-w-sm"
              label="Method"
              options={takeMethods(app)}
              value={method}
              onChange={setMethod}
            />
            <p className="mt-2 text-xs text-muted">
              {method === TAKE_ROLL_CALL
                ? 'You read the roster and tap Present or Absent for each student. No location is needed.'
                : app
                  ? 'Students mark themselves from their registered phone, or with the code on the board.'
                  : 'You show the 4-digit code and your browser shares your position, so students in the room can mark themselves.'}
            </p>
            {method !== TAKE_ROLL_CALL && (
              <label className="mt-3 flex items-start gap-2.5 text-sm text-ink">
                <input
                  type="checkbox"
                  checked={codeOnly}
                  onChange={(event) => setCodeOnly(event.target.checked)}
                  className="mt-0.5 size-4 shrink-0 accent-brand-600"
                />
                <span>
                  <span className="block font-semibold">{CODE_ONLY_LABEL}</span>
                  <span className="mt-0.5 block text-xs text-muted">
                    For when location is not available. Marks are saved as Code and flagged No
                    location in your list and the Excel sheet so you can review them.
                  </span>
                </span>
              </label>
            )}
          </div>
          <SessionTiming
            timing={timing}
            onChange={setTiming}
            rollCall={method === TAKE_ROLL_CALL}
            problem={picked ? problem : ''}
          />
          <div>
            <Button loading={busy} disabled={!picked || Boolean(problem)} onClick={onStart}>
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
                <span className="block wrap-long text-sm font-semibold text-ink">
                  {dots(session.course_code, session.course_title)}
                </span>
                <span className="mt-0.5 block wrap-long text-xs text-muted">
                  {dots(
                    session.batch,
                    sectionLabel(session.section),
                    formatDate(session.started_at),
                    formatClock(session.started_at),
                  )}
                </span>
              </span>
              <span className="shrink-0 text-sm font-semibold text-ink">
                {presentOfRoster(session)}
                {session.late > 0 ? ` · ${session.late} L` : ''}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </Card>
  )
}

function ClassList({ classes, loading, onExport, onDelete, busy }) {
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
              <span className="block wrap-long text-sm font-semibold text-ink">
                {dots(entry.course_code, entry.course_title)}
              </span>
              <span className="mt-0.5 block wrap-long text-xs text-muted">
                {dots(
                  entry.batch,
                  sectionLabel(entry.section),
                  `${entry.sessions} session${entry.sessions === 1 ? '' : 's'}`,
                  `${entry.below_fifty} below 50 %`,
                )}
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
            <span className="flex flex-wrap gap-2">
              <Button
                variant="secondary"
                size="sm"
                loading={busy === classKey(entry)}
                onClick={() => onExport(entry)}
              >
                <Download size={15} /> Excel
              </Button>
              <Button
                variant="danger"
                size="sm"
                loading={busy === `delete-${classKey(entry)}`}
                onClick={() => onDelete(entry)}
              >
                <Trash2 size={15} /> Delete
              </Button>
            </span>
          </li>
        ))}
      </ul>
    </Card>
  )
}

export default function TeacherAttendancePage() {
  const toast = useToast()
  const confirm = useConfirm()
  const now = useNow(5000)
  const [tab, setTab] = useQueryTab(TAB_VALUES, TEACHER_TAB_TAKE)
  const [timing, setTiming] = useState(() => newTiming(DEFAULT_MINUTES))
  const timingError = timingProblem(timing, new Date(now))
  const [method, setMethod] = useState(TAKE_BLUETOOTH)
  const [codeOnly, setCodeOnly] = useState(false)
  const [picked, setPicked] = useState('')
  const [busy, setBusy] = useState('')
  const [open, setOpen] = useState(null)
  const [marks, setMarks] = useState([])
  const [detailID, setDetailID] = useState(0)
  const [status, setStatus] = useState(null)
  const broadcasting = useRef(0)
  const asked = useRef(false)
  const app = beaconSupported()
  const options = useResource('/api/teacher/attendance/options')
  const optionList = options.data
  useEffect(() => {
    if (!picked || !optionList) return
    const entry = (optionList.today || [])
      .concat(optionList.classes || [])
      .find((item) => classKey(item) === picked)
    if (!entry) return
    const hours = entry.hours || hoursForClass(entry.start_time, entry.end_time)
    setTiming((current) => ({ ...current, hours }))
  }, [picked, optionList])
  const history = useResource('/api/teacher/attendance/sessions')
  const classes = useResource('/api/teacher/attendance/classes')
  useEffect(() => {
    if (options.data?.open) setOpen(options.data.open)
  }, [options.data])
  const refreshSession = useCallback(async (sessionID) => {
    if (!sessionID) return
    try {
      const data = await api.get(`/api/teacher/attendance/sessions/${sessionID}`)
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
  }, [])
  const refreshOpen = useCallback(() => refreshSession(open?.id), [refreshSession, open?.id])
  usePolling(open?.id && open?.method !== TAKE_ROLL_CALL ? refreshOpen : null, 5000)
  const sendPlace = useCallback(
    async (position) => {
      if (!open?.id) return
      const fresh = await api.post(`/api/teacher/attendance/sessions/${open.id}/location`, position)
      if (fresh?.status === 'open') setOpen(fresh)
    },
    [open?.id],
  )
  const watching = !app && Boolean(open?.id) && open?.method !== TAKE_ROLL_CALL && !open?.code_only
  const { place, locate } = useWebLocation({
    active: watching,
    every: 20000,
    onPosition: sendPlace,
  })
  const readStatus = useCallback(async () => {
    if (!beaconSupported()) return null
    try {
      const fresh = await beaconStatus()
      const seen = { ...fresh, asked: asked.current }
      setStatus(seen)
      return seen
    } catch {
      return null
    }
  }, [])
  usePolling(
    app && open?.id && open?.method !== TAKE_ROLL_CALL && !open?.code_only ? readStatus : null,
    4000,
  )
  const prepare = useCallback(
    async (session) => {
      if (!beaconSupported() || session?.code_only) return
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
    if (what === 'allow' || what === 'retry') {
      setBusy('locate')
      try {
        await locate()
      } finally {
        setBusy('')
      }
      return
    }
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
  const switchCodeOnly = async () => {
    const ok = await confirm({
      title: 'Switch to code only?',
      message:
        'Students are marked with the 4-digit code alone and their position is not checked. Every such mark is flagged No location for you to review.',
      confirmLabel: 'Switch',
    })
    if (!ok) return
    setBusy('code-only')
    try {
      const session = await api.post(`/api/teacher/attendance/sessions/${open.id}/code-only`)
      if (broadcasting.current && beaconSupported()) await stopBeacon().catch(() => null)
      broadcasting.current = 0
      setOpen(session)
      toast.success('Students now mark with the code only.')
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  useEffect(() => {
    if (!app || !open?.id || open.method === TAKE_ROLL_CALL || open.code_only) return
    if (!open.token_key) {
      toast.error('This session did not come with its Bluetooth key. Close it and start again.')
      return
    }
    if (broadcasting.current === open.id) return
    prepare(open)
  }, [app, open, prepare, toast])
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
      const device = await phoneId().catch(() => '')
      const session = await api.post(
        '/api/teacher/attendance/sessions',
        startBody(entry, timingBody(timing), method, codeOnly, device),
      )
      setOpen(session)
      setMarks([])
      await refreshSession(session.id)
      toast.success(method === TAKE_ROLL_CALL ? 'The roll call is open.' : 'Attendance is open.')
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
  const removeSheet = async (entry) => {
    const ok = await confirm({
      title: 'Delete the attendance sheet?',
      message: deleteSheetQuestion(entry),
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setBusy(`delete-${classKey(entry)}`)
    try {
      await api.delete('/api/teacher/attendance/sheet', {
        course_id: entry.course_id,
        batch: entry.batch,
        section: entry.section,
      })
      toast.success('The attendance sheet is deleted.')
      classes.reload()
      history.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const close = async () => {
    setBusy('close')
    try {
      const result = await api.post(`/api/teacher/attendance/sessions/${open.id}/close`)
      if (beaconSupported()) await stopBeacon().catch(() => null)
      broadcasting.current = 0
      setStatus(null)
      setOpen(null)
      setMarks([])
      toast.success(closedMessage(result))
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
        tabs={<StatusTabs tabs={TABS} value={tab} onChange={setTab} tone="header" />}
      />
      {options.error && (
        <Alert tone="error" className="mb-5">
          {options.error}
        </Alert>
      )}
      {tab === TEACHER_TAB_TAKE && (
        <div className="flex flex-col gap-5">
          {open && open.method === TAKE_ROLL_CALL ? (
            <RollCall
              session={open}
              marks={marks}
              busy={busy}
              onReload={refreshOpen}
              onFinish={close}
            />
          ) : open ? (
            <OpenSession
              session={open}
              marks={marks}
              now={now}
              status={status}
              place={place}
              app={app}
              busy={busy}
              onExtend={extend}
              onClose={close}
              onFix={fix}
              onCodeOnly={switchCodeOnly}
            />
          ) : options.loading ? (
            <SkeletonCard />
          ) : (
            <StartPanel
              options={options.data}
              timing={timing}
              setTiming={setTiming}
              problem={timingError}
              picked={picked}
              setPicked={setPicked}
              method={method}
              setMethod={setMethod}
              codeOnly={codeOnly}
              setCodeOnly={setCodeOnly}
              app={app}
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
          onDelete={removeSheet}
        />
      )}
      {tab === TEACHER_TAB_PHONES && <PhoneRequestList />}
      <SessionSheet
        sessionId={detailID}
        onClose={() => setDetailID(0)}
        onChanged={() => {
          history.reload()
          classes.reload()
        }}
        onDeleted={() => {
          history.reload()
          classes.reload()
          options.reload()
        }}
      />
    </>
  )
}
