import { attended } from '../../lib/attendanceTiming'
import { useEffect, useState } from 'react'
import StudentRow from './StudentRow'
import { Button, Sheet, useConfirm, useToast } from '../ui'
import api from '../../lib/api'
import { formatClock, formatDate } from '../../lib/time'
import { markMeta, methodLabel, presentOfRoster, sectionLabel, sortedRoster } from '../../lib/attendance'
import { dots } from '../../lib/dots'

const metaOf = (mark, take) =>
  dots(
    methodLabel(mark.method, take),
    markMeta(mark, formatClock),
    mark.edited_by ? `Edited by ${mark.edited_by}` : '',
    mark.edited_by && mark.edited_at ? formatDate(mark.edited_at) : '',
  )

export default function SessionSheet({ sessionId, onClose, onChanged, onDeleted }) {
  const toast = useToast()
  const confirm = useConfirm()
  const [removing, setRemoving] = useState(false)
  const [detail, setDetail] = useState(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(0)
  useEffect(() => {
    if (!sessionId) return undefined
    let active = true
    setDetail(null)
    setError('')
    api
      .get(`/api/teacher/attendance/sessions/${sessionId}`)
      .then((data) => {
        if (active) setDetail(data)
      })
      .catch((failure) => {
        if (active) setError(failure.message)
      })
    return () => {
      active = false
    }
  }, [sessionId])
  const save = async (studentId, status, note) => {
    setBusy(studentId)
    try {
      const data = await api.patch(
        `/api/teacher/attendance/sessions/${sessionId}/students/${studentId}`,
        { status, note },
      )
      setDetail(data)
      toast.success('The attendance of this student is updated.')
      onChanged?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const remove = async () => {
    const ok = await confirm({
      title: 'Delete this attendance session?',
      message: 'The marks of this class are removed for everybody. This cannot be undone.',
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setRemoving(true)
    try {
      await api.delete(`/api/teacher/attendance/sessions/${sessionId}`)
      toast.success('The attendance session is deleted.')
      onDeleted?.()
      onClose()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setRemoving(false)
    }
  }
  const session = detail?.session
  const marks = detail?.marks || []
  const roster = sortedRoster(marks)
  const present = marks.filter((mark) => attended(mark.status)).length
  const late = marks.filter((mark) => mark.status === 'late').length
  return (
    <Sheet
      open={Boolean(sessionId)}
      onClose={onClose}
      title={session ? `${session.course_code} · ${session.batch}` : 'Attendance'}
      description={
        session
          ? dots(
              sectionLabel(session.section),
              formatDate(session.started_at),
              `${presentOfRoster({ present, roster: marks.length })} present`,
              late > 0 ? `${late} late` : '',
              session.hours > 1 ? `${session.hours} class hours` : '',
            )
          : ''
      }
      full
      footer={
        detail ? (
          <Button variant="danger" loading={removing} onClick={remove}>
            Delete this session
          </Button>
        ) : null
      }
    >
      {error && <p className="text-sm text-danger-600">{error}</p>}
      {!detail && !error && <p className="text-sm text-muted">Loading the class…</p>}
      {detail && marks.length === 0 && (
        <p className="text-sm text-muted">This class has no students on its roster.</p>
      )}
      <ul className="flex flex-col">
        {roster.map((mark) => (
          <StudentRow
            key={mark.student_id}
            mark={mark}
            status={mark.status}
            meta={metaOf(mark, session?.method)}
            busy={busy === mark.student_id}
            onMark={(status, note) => save(mark.student_id, status, note)}
            onSaveNote={(note) => save(mark.student_id, mark.status || 'absent', note)}
          />
        ))}
      </ul>
    </Sheet>
  )
}
