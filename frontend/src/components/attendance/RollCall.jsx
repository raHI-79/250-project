import { useState } from 'react'
import { ListChecks } from 'lucide-react'
import StudentRow from './StudentRow'
import { Alert, Button, Card, CardHeader, useConfirm, useToast } from '../ui'
import api from '../../lib/api'
import { sectionLabel, sortedRoster } from '../../lib/attendance'

export default function RollCall({ session, marks, busy, onReload, onFinish }) {
  const toast = useToast()
  const confirm = useConfirm()
  const [working, setWorking] = useState(0)
  const roster = sortedRoster(marks)
  const present = marks.filter((entry) => entry.status === 'present').length
  const left = marks.filter((entry) => !entry.marked_at && entry.method !== 'manual').length
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
        api.patch(`/api/teacher/attendance/sessions/${session.id}/students/${studentId}/note`, {
          note: text,
        }),
      'Note saved',
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
  const statusOf = (entry) => {
    const marked = Boolean(entry.marked_at) || entry.method === 'manual'
    return marked ? entry.status : ''
  }
  return (
    <Card hover={false}>
      <CardHeader
        title={`Roll call · ${session.course_code}`}
        subtitle={`${session.batch} · ${sectionLabel(session.section)} · ${present}/${marks.length} present`}
        className="px-0 pt-0"
      />
      {left > 0 && (
        <Alert tone="info" className="mb-4">
          {left} student{left === 1 ? ' is' : 's are'} still unmarked. Finish marks them absent.
        </Alert>
      )}
      <div className="flex flex-wrap gap-2">
        <Button variant="secondary" size="sm" loading={working === -1} onClick={markAll}>
          <ListChecks size={15} />
          Mark all present
        </Button>
        <Button variant="danger" size="sm" loading={busy === 'close'} onClick={finish}>
          Finish
        </Button>
      </div>
      {roster.length === 0 ? (
        <p className="mt-4 text-sm text-muted">This class has no students on its roster.</p>
      ) : (
        <ul className="mt-3 flex flex-col">
          {roster.map((entry) => (
            <StudentRow
              key={entry.student_id}
              mark={entry}
              status={statusOf(entry)}
              busy={working === entry.student_id}
              onMark={(status, text) => mark(entry.student_id, status, text)}
              onSaveNote={(text) => note(entry.student_id, text)}
            />
          ))}
        </ul>
      )}
    </Card>
  )
}
