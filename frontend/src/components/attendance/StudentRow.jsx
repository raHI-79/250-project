import { useEffect, useRef, useState } from 'react'
import { Check, NotebookPen, X } from 'lucide-react'
import cx from '../../lib/cx'
import { markFlags, rollNumber } from '../../lib/attendance'

const MARKS = {
  present: 'bg-success-600 text-white',
  absent: 'bg-danger-600 text-white',
  idle: 'border border-line-strong bg-surface text-ink-soft',
}

function MarkButton({ tone, label, short, icon: Icon, active, busy, onClick }) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={active}
      disabled={busy}
      onClick={onClick}
      className={cx(
        'press inline-flex h-11 min-w-11 items-center justify-center gap-1 rounded-xl px-2.5 text-sm font-bold disabled:opacity-60',
        active ? MARKS[tone] : MARKS.idle,
      )}
    >
      <Icon size={17} strokeWidth={2.4} />
      {short}
    </button>
  )
}

export default function StudentRow({ mark, status, busy, meta, onMark, onSaveNote }) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(mark.note || '')
  const field = useRef(null)
  useEffect(() => {
    setDraft(mark.note || '')
  }, [mark.note])
  useEffect(() => {
    if (editing) field.current?.focus()
  }, [editing])
  const roll = rollNumber(mark.reg_no)
  const save = async () => {
    await onSaveNote(draft)
    setEditing(false)
  }
  return (
    <li className="border-b border-line py-2 last:border-b-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <span
          aria-hidden={!roll}
          className="grid h-11 w-12 shrink-0 place-items-center rounded-xl bg-brand-50 text-base font-extrabold text-brand-700 tabular-nums"
        >
          {roll}
        </span>
        <span className="min-w-[8rem] flex-1">
          <span className="block wrap-long text-sm leading-snug font-semibold text-ink">
            {mark.full_name}
          </span>
          <span className="block wrap-long text-xs text-muted tabular-nums">{mark.reg_no}</span>
          {meta && <span className="block wrap-long text-2xs text-faint">{meta}</span>}
          {markFlags(mark).map((flag) => (
            <span
              key={flag}
              className="mt-0.5 inline-block rounded-full bg-warning-50 px-2 py-0.5 text-2xs font-semibold text-warning-700"
            >
              {flag}
            </span>
          ))}
          {mark.note && !editing && (
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="block max-w-full text-left wrap-long text-xs text-ink-soft italic hover:text-brand-700"
            >
              {mark.note}
            </button>
          )}
        </span>
        <span className="ml-auto flex shrink-0 items-center gap-1.5">
          <MarkButton
            tone="present"
            label={`Mark ${mark.full_name} present`}
            short="P"
            icon={Check}
            active={status === 'present'}
            busy={busy}
            onClick={() => onMark('present', mark.note || '')}
          />
          <MarkButton
            tone="absent"
            label={`Mark ${mark.full_name} absent`}
            short="A"
            icon={X}
            active={status === 'absent'}
            busy={busy}
            onClick={() => onMark('absent', mark.note || '')}
          />
          <button
            type="button"
            aria-label={mark.note ? 'Edit the note' : 'Add a note'}
            aria-expanded={editing}
            onClick={() => setEditing((current) => !current)}
            className={cx(
              'press grid size-11 shrink-0 place-items-center rounded-xl border border-line-strong',
              mark.note ? 'bg-brand-50 text-brand-700' : 'bg-surface text-muted',
            )}
          >
            <NotebookPen size={17} />
          </button>
        </span>
      </div>
      {editing && (
        <div className="mt-2 flex flex-wrap items-center gap-2 pl-15">
          <input
            ref={field}
            type="text"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') save()
              if (event.key === 'Escape') setEditing(false)
            }}
            aria-label={`Note for ${mark.full_name}`}
            placeholder="Why this student is present or absent"
            className="h-11 min-w-0 flex-1 rounded-xl border border-line-strong bg-wash px-3 text-sm text-ink outline-none focus:border-brand-600"
          />
          <button
            type="button"
            disabled={busy}
            onClick={save}
            className="press h-11 shrink-0 rounded-xl bg-brand-600 px-4 text-sm font-semibold text-white disabled:opacity-60"
          >
            Save
          </button>
        </div>
      )}
    </li>
  )
}
