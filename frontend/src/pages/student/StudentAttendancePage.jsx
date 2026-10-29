import { useCallback, useEffect, useRef, useState } from 'react'
import { Bluetooth, CheckCircle2, KeyRound, Smartphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Field,
  Input,
  PageHeader,
  SkeletonCard,
  Textarea,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { useResource } from '../../lib/hooks'
import { formatClock } from '../../lib/time'
import { isAndroidApp } from '../../lib/native'
import { paths } from '../../lib/paths'
import {
  askMark,
  confirmIdentity,
  listenForToken,
  markStatus,
  phoneId,
  readPosition,
  sectionLabel,
  turnMarkBluetoothOn,
} from '../../lib/attendance'

const WEB_MESSAGE =
  'Attendance is marked from the Android app. Open the app and tap I am present.'

const STEP_IDLE = 'idle'

const STEP_LISTENING = 'listening'

const STEP_READY = 'ready'

const STEP_CODE = 'code'

function PhonePanel({ phone, busy, onRegister, onRequest }) {
  const [reason, setReason] = useState('')
  if (!phone.registered) {
    return (
      <Card hover={false}>
        <p className="label-caps">Your attendance phone</p>
        <p className="mt-2 text-sm text-ink-soft">
          Attendance is tied to one phone so nobody can mark a classmate. Register this phone to
          start marking yourself present.
        </p>
        <Button className="mt-4" loading={busy === 'register'} onClick={onRegister}>
          <Smartphone size={16} /> Use this phone
        </Button>
      </Card>
    )
  }
  if (phone.this_phone) return null
  if (phone.request_status === 'pending') {
    return (
      <Alert tone="info" title="Your teachers were asked">
        You asked to use this phone. Any teacher of your batch can approve it, and you can mark
        attendance once one of them does.
      </Alert>
    )
  }
  return (
    <Card hover={false}>
      <p className="label-caps">This is not your registered phone</p>
      <p className="mt-2 text-sm text-ink-soft">
        {phone.request_status === 'rejected' && phone.decider_name
          ? `${phone.decider_name} refused your last request${phone.decision_note ? `: ${phone.decision_note}` : '.'}`
          : 'Tell your teachers why you need this phone. Any teacher of your batch can approve it.'}
      </p>
      {phone.changes >= phone.limit && (
        <p className="mt-2 text-xs text-warning-700">
          You already changed your phone {phone.changes} times this semester, so a teacher has to
          add a note to approve another change.
        </p>
      )}
      <Field label="Why you need this phone" htmlFor="phone-reason" className="mt-4">
        <Textarea
          id="phone-reason"
          rows={3}
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="My old phone broke and this is my new one"
        />
      </Field>
      <Button
        className="mt-3"
        loading={busy === 'request'}
        disabled={!reason.trim()}
        onClick={() => onRequest(reason.trim())}
      >
        Request to use this phone
      </Button>
    </Card>
  )
}

export default function StudentAttendancePage() {
  const toast = useToast()
  const [device, setDevice] = useState('')
  const [step, setStep] = useState(STEP_IDLE)
  const [token, setToken] = useState('')
  const [code, setCode] = useState('')
  const [hint, setHint] = useState('')
  const [busy, setBusy] = useState('')
  const [done, setDone] = useState(false)
  const listening = useRef(false)
  const app = isAndroidApp()
  useEffect(() => {
    if (!app) return
    phoneId()
      .then((id) => setDevice(id))
      .catch(() => setDevice(''))
  }, [app])
  const { data, loading, error, reload } = useResource(
    app && !device ? '' : '/api/students/attendance/open',
    device ? { device_id: device } : undefined,
  )
  const session = data?.session
  const phone = data?.phone || { registered: false, this_phone: false, changes: 0, limit: 2 }
  const canMark = app && phone.registered && phone.this_phone && session && !session.present
  const listen = useCallback(async () => {
    if (listening.current) return
    listening.current = true
    setStep(STEP_LISTENING)
    setHint('')
    try {
      const asked = await askMark()
      if (!asked?.bluetooth_on) {
        setHint('Bluetooth is off. Turn it on so your phone can hear the class.')
        setStep(STEP_IDLE)
        return
      }
      if (!asked?.nearby || !asked?.place) {
        setHint('Allow nearby devices and precise location, then try again.')
        setStep(STEP_IDLE)
        return
      }
      const heard = await listenForToken(10)
      if (heard?.token) {
        setToken(heard.token)
        setStep(STEP_READY)
        return
      }
      setToken('')
      setStep(STEP_CODE)
      setHint('Nothing was heard from the class. Enter the code on the board.')
    } catch (failure) {
      setStep(STEP_IDLE)
      setHint(failure.message || 'Bluetooth could not be used.')
    } finally {
      listening.current = false
    }
  }, [])
  useEffect(() => {
    if (!canMark) return
    listen()
  }, [canMark, listen])
  const fixBluetooth = async () => {
    try {
      await turnMarkBluetoothOn()
      const status = await markStatus()
      if (status?.bluetooth_on) listen()
    } catch (failure) {
      toast.error(failure.message)
    }
  }
  const mark = async (useCode) => {
    setBusy('mark')
    try {
      const confirmed = await confirmIdentity()
      if (confirmed?.available && !confirmed?.confirmed) {
        toast.error(confirmed.message || 'Confirm it is you to be marked present.')
        return
      }
      const place = await readPosition()
      if (!place?.found) {
        toast.error('Your phone did not report a position. Turn location on and try again.')
        return
      }
      await api.post('/api/students/attendance/mark', {
        session_id: session.id,
        token: useCode ? '' : token,
        code: useCode ? code.trim() : '',
        lat: place.lat,
        lng: place.lng,
        accuracy_m: place.accuracy_m,
        is_mock: Boolean(place.is_mock),
        device_id: device,
      })
      setDone(true)
      toast.success('You are marked present.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const register = async () => {
    setBusy('register')
    try {
      const result = await api.post('/api/students/attendance/phone', { device_id: device })
      toast.success(result?.message || 'This phone is now your attendance phone.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const request = async (reason) => {
    setBusy('request')
    try {
      const result = await api.post('/api/students/attendance/phone-request', {
        device_id: device,
        reason,
      })
      toast.success(result?.message || 'Your teachers were asked.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  if (!app) {
    return (
      <>
        <PageHeader eyebrow="Your classes" title="Attendance" />
        <EmptyState
          icon={Smartphone}
          title="Use the Android app"
          text={WEB_MESSAGE}
          action={
            <Button as="a" href={paths.downloadAndroid()} variant="secondary">
              Get the app
            </Button>
          }
        />
      </>
    )
  }
  return (
    <>
      <PageHeader
        eyebrow="Your classes"
        title="Attendance"
        subtitle="One tap from your own phone, inside the classroom, while the teacher keeps it open."
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      <div className="flex flex-col gap-5">
        <PhonePanel phone={phone} busy={busy} onRegister={register} onRequest={request} />
        {loading ? (
          <SkeletonCard />
        ) : !session ? (
          <EmptyState
            icon={CheckCircle2}
            title="No attendance is open"
            text="When a teacher opens attendance for one of your courses, this screen lets you mark yourself present."
          />
        ) : (
          <Card hover={false} rule>
            <p className="label-caps">Open now</p>
            <p className="font-display mt-1 text-section font-bold text-ink">
              {session.course_code} · {session.course_title}
            </p>
            <p className="mt-1 text-xs text-muted">
              {session.teacher_name} · {session.batch} · {sectionLabel(session.section)} · closes{' '}
              {formatClock(session.ends_at)}
            </p>
            {session.present || done ? (
              <Alert tone="success" className="mt-4" title="You are marked present">
                Nothing else to do for this class.
              </Alert>
            ) : !phone.registered || !phone.this_phone ? (
              <Alert tone="warning" className="mt-4">
                Register this phone, or ask your teachers for it, before you can mark attendance.
              </Alert>
            ) : (
              <div className="mt-4 flex flex-col gap-3">
                {hint && (
                  <Alert tone="warning">
                    {hint}
                    <Button variant="ghost" size="sm" className="mt-2" onClick={fixBluetooth}>
                      Turn Bluetooth on
                    </Button>
                  </Alert>
                )}
                <div
                  className={cx(
                    'flex items-center gap-2 text-sm font-semibold',
                    step === STEP_READY ? 'text-success-700' : 'text-muted',
                  )}
                >
                  <Bluetooth size={16} />
                  {step === STEP_LISTENING
                    ? 'Listening for your class…'
                    : step === STEP_READY
                      ? 'Your class was heard'
                      : 'Not heard yet'}
                </div>
                <Button
                  size="lg"
                  loading={busy === 'mark'}
                  disabled={step !== STEP_READY}
                  onClick={() => mark(false)}
                >
                  <CheckCircle2 size={18} /> I am present
                </Button>
                {step === STEP_CODE || step === STEP_IDLE ? (
                  <div className="rounded-card border border-line bg-wash p-4">
                    <p className="label-caps flex items-center gap-1.5">
                      <KeyRound size={13} /> Enter the code on the board
                    </p>
                    <div className="mt-2 flex items-end gap-2">
                      <Field label="Code" htmlFor="attendance-code" className="flex-1">
                        <Input
                          id="attendance-code"
                          inputMode="numeric"
                          maxLength={4}
                          value={code}
                          onChange={(event) =>
                            setCode(event.target.value.replace(/\D/g, '').slice(0, 4))
                          }
                          placeholder="0000"
                        />
                      </Field>
                      <Button
                        loading={busy === 'mark'}
                        disabled={code.trim().length !== 4}
                        onClick={() => mark(true)}
                      >
                        Mark me
                      </Button>
                    </div>
                  </div>
                ) : (
                  <Button variant="ghost" size="sm" onClick={() => setStep(STEP_CODE)}>
                    Enter the code on the board
                  </Button>
                )}
                <Button variant="secondary" size="sm" onClick={listen}>
                  Listen again
                </Button>
              </div>
            )}
          </Card>
        )}
      </div>
    </>
  )
}
