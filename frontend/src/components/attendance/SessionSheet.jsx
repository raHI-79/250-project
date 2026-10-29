import { useEffect, useState } from 'react'
import { Check, X } from 'lucide-react'
import { Button, Field, Sheet, Textarea, useToast } from '../ui'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { formatClock, formatDate } from '../../lib/time'
import { sectionLabel } from '../../lib/attendance'

const methodLabel = (method) => {
  if (method === 'app') return 'App'
  if (method === 'code') return 'Code'
  if (method === 'manual') return 'By teacher'
  if (method === 'system') return 'Not marked'
  return ''
}

function MarkRow({ mark, open, onOpen, onSave, busy }) {
  const [status, setStatus] = useState(mark.status || 'absent')
  const [note, setNote] = useState(mark.note || '')
  useEffect(() => {
    setStatus(mark.status || 'absent')
    setNote(mark.note || '')
  }, [mark.status, mark.note])
  const present = mark.status === 'present'
  return (
    <li className="border-b border-line last:border-b-0">
      <button
        type="button"
        onClick={onOpen}
        className="press flex w-full items-center gap-3 px-1 py-3 text-left transition-colors hover:bg-wash"
      >
        <span
          className={cx(
            'grid size-9 shrink-0 place-items-center rounded-full',
            present ? 'bg-success-50 text-success-600' : 'bg-danger-50 text-danger-600',
          )}
        >
          {present ? <Check size={17} /> : <X size={17} />}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold text-ink">{mark.full_name}</span>
          <span className="mt-0.5 block truncate text-xs text-muted">
            {mark.reg_no} · {methodLabel(mark.method)}
            {mark.marked_at ? ` · ${formatClock(mark.marked_at)}` : ''}
            {mark.distance_m ? ` · ${mark.distance_m} m` : ''}
          </span>
          {mark.note && <span className="mt-0.5 block text-xs text-ink-soft">{mark.note}</span>}
          {mark.edited_by && (
            <span className="mt-0.5 block text-2xs text-faint">
              Edited by {mark.edited_by}
              {mark.edited_at ? ` · ${formatDate(mark.edited_at)}` : ''}
            </span>
          )}
        </span>
      </button>
      {open && (
        <div className="flex flex-col gap-3 px-1 pb-4">
          <div className="flex gap-2">
            <Button
              variant={status === 'present' ? 'primary' : 'secondary'}
              size="sm"
              onClick={() => setStatus('present')}
            >
              Present
            </Button>
            <Button
              variant={status === 'absent' ? 'danger' : 'secondary'}
              size="sm"
              onClick={() => setStatus('absent')}
            >
              Absent
            </Button>
          </div>
          <Field label="Note" htmlFor={`note-${mark.student_id}`}>
            <Textarea
              id={`note-${mark.student_id}`}
              rows={2}
              value={note}
              onChange={(event) => setNote(event.target.value)}
              placeholder="Why this student is present or absent"
            />
          </Field>
          <div>
            <Button size="sm" loading={busy} onClick={() => onSave(status, note)}>
              Save
            </Button>
          </div>
        </div>
      )}
    </li>
  )
}

export default function SessionSheet({ sessionId, onClose, onChanged }) {
  const toast = useToast()
  const [detail, setDetail] = useState(null)
  const [error, setError] = useState('')
  const [open, setOpen] = useState(0)
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
      setOpen(0)
      toast.success('The attendance of this student is updated.')
      onChanged?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const session = detail?.session
  const marks = detail?.marks || []
  const present = marks.filter((mark) => mark.status === 'present').length
  return (
    <Sheet
      open={Boolean(sessionId)}
      onClose={onClose}
      title={session ? `${session.course_code} · ${session.batch}` : 'Attendance'}
      description={
        session
          ? `${sectionLabel(session.section)} · ${formatDate(session.started_at)} · ${present}/${marks.length} present`
          : ''
      }
      full
    >
      {error && <p className="text-sm text-danger-600">{error}</p>}
      {!detail && !error && <p className="text-sm text-muted">Loading the class…</p>}
      {detail && marks.length === 0 && (
        <p className="text-sm text-muted">This class has no students on its roster.</p>
      )}
      <ul className="flex flex-col">
        {marks.map((mark) => (
          <MarkRow
            key={mark.student_id}
            mark={mark}
            open={open === mark.student_id}
            busy={busy === mark.student_id}
            onOpen={() => setOpen(open === mark.student_id ? 0 : mark.student_id)}
            onSave={(status, note) => save(mark.student_id, status, note)}
          />
        ))}
      </ul>
    </Sheet>
  )
}
