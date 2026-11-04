import { useEffect, useState } from 'react'
import { Check, ListChecks, X } from 'lucide-react'
import Avatar from '../ui/Avatar'
import { Alert, Button, Card, CardHeader, Field, Textarea, useConfirm, useToast } from '../ui'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { sectionLabel } from '../../lib/attendance'

function RollRow({ mark, busy, onMark, onNote }) {
  const [note, setNote] = useState(mark.note || '')
  const [saved, setSaved] = useState(false)
  useEffect(() => {
    setNote(mark.note || '')
    setSaved(false)
  }, [mark.note])
  const marked = Boolean(mark.marked_at) || mark.method === 'manual'
  const present = marked && mark.status === 'present'
  const absent = marked && mark.status === 'absent'
  return (
    <li className="flex flex-col gap-3 border-b border-line py-3 last:border-b-0">
      <div className="flex flex-wrap items-center gap-3">
        <Avatar name={mark.full_name} src={mark.photo_url} size="sm" />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold text-ink">{mark.full_name}</span>
          <span className="block truncate text-xs text-muted">{mark.reg_no}</span>
        </span>
        <span className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant={present ? 'primary' : 'secondary'}
            loading={busy}
            onClick={() => onMark('present', note)}
          >
            <Check size={14} />
            Present
          </Button>
          <Button
            size="sm"
            variant={absent ? 'danger' : 'secondary'}
            loading={busy}
            onClick={() => onMark('absent', note)}
          >
            <X size={14} />
            Absent
          </Button>
        </span>
      </div>
      <div className="flex flex-col gap-2">
        <Field label="Note" htmlFor={`roll-note-${mark.student_id}`}>
          <Textarea
            id={`roll-note-${mark.student_id}`}
            rows={2}
            value={note}
            onChange={(event) => {
              setNote(event.target.value)
              setSaved(false)
            }}
            placeholder="Why this student is present or absent"
          />
        </Field>
        <div className="flex flex-wrap items-center gap-3">
          <Button
            size="sm"
            variant="secondary"
            loading={busy}
            onClick={async () => {
              await onNote(note)
              setSaved(true)
            }}
          >
            Save note
          </Button>
          {saved && <span className="text-xs font-semibold text-success-700">Note saved</span>}
        </div>
      </div>
    </li>
  )
}

export default function RollCall({ session, marks, busy, onReload, onFinish }) {
  const toast = useToast()
  const confirm = useConfirm()
  const [working, setWorking] = useState(0)
  const present = marks.filter((mark) => mark.status === 'present').length
  const left = marks.filter((mark) => !mark.marked_at && mark.method !== 'manual').length
  const run = async (studentId, call, message) => {
    setWorking(studentId)
    try {
      await call()
      if (message) toast.success(message)
      onReload?.()
    } catch (failure) {
      toast.error(failure.message)
      throw failure
    } finally {
      setWorking(0)
    }
  }
  const mark = (studentId, status, note) =>
    run(
      studentId,
      () =>
        api.patch(`/api/teacher/attendance/sessions/${session.id}/students/${studentId}`, {
          status,
          note,
        }),
      status === 'present' ? 'Marked present' : 'Marked absent',
    ).catch(() => null)
  const note = (studentId, text) =>
    run(
      studentId,
      () =>
        api.patch(
          `/api/teacher/attendance/sessions/${session.id}/students/${studentId}/note`,
          { note: text },
        ),
      '',
    ).catch(() => null)
  const markAll = () =>
    run(
      -1,
      () => api.post(`/api/teacher/attendance/sessions/${session.id}/mark-all`),
      'Everybody on the roster is present',
    ).catch(() => null)
  const finish = async () => {
    const ok = await confirm({
      title: 'Finish the roll call?',
      message:
        left > 0
          ? `${left} student${left === 1 ? '' : 's'} are still unmarked and will be saved as absent.`
          : 'The sheet is saved and the class is closed.',
      confirmLabel: 'Finish',
    })
    if (!ok) return
    onFinish()
  }
  return (
    <Card hover={false}>
      <CardHeader
        title={`Roll call · ${session.course_code}`}
        subtitle={`${session.batch} · ${sectionLabel(session.section)} · ${present}/${marks.length} present`}
        className="px-0 pt-0"
      />
      <div className="flex flex-wrap gap-2">
        <Button variant="secondary" size="sm" loading={working === -1} onClick={markAll}>
          <ListChecks size={15} />
          Mark all present
        </Button>
        <Button variant="danger" size="sm" loading={busy === 'close'} onClick={finish}>
          Finish
        </Button>
      </div>
      {left > 0 && (
        <Alert tone="info" className="mt-4">
          {left} student{left === 1 ? ' is' : 's are'} still unmarked. Finish marks them absent.
        </Alert>
      )}
      {marks.length === 0 ? (
        <p className={cx('mt-4 text-sm text-muted')}>This class has no students on its roster.</p>
      ) : (
        <ul className="mt-4 flex flex-col">
          {marks.map((entry) => (
            <RollRow
              key={entry.student_id}
              mark={entry}
              busy={working === entry.student_id}
              onMark={(status, text) => mark(entry.student_id, status, text)}
              onNote={(text) => note(entry.student_id, text)}
            />
          ))}
        </ul>
      )}
    </Card>
  )
}
