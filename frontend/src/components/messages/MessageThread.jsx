import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { MessagesSquare, Pencil, Trash2 } from 'lucide-react'
import AttachmentView from '../AttachmentView'
import { Avatar, Button, EmptyState, SkeletonLines } from '../ui'
import cx from '../../lib/cx'
import { paths } from '../../lib/paths'
import { EDIT_WINDOW, MAX_MESSAGE_BODY, shortRoleLabel } from '../../lib/messages'
import { dayLabel, formatClock } from '../../lib/time'

const TICK = 10000

const GROUP_WINDOW = 5 * 60 * 1000

const dayKey = (value) => String(value || '').slice(0, 10)

const clockOf = (value) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return formatClock(
    `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`,
  )
}

const byDay = (items) => {
  const days = []
  items.forEach((item) => {
    const key = dayKey(item.created_at)
    const last = days[days.length - 1]
    if (last && last.key === key) last.items.push(item)
    else days.push({ key, items: [item] })
  })
  return days
}

export default function MessageThread({ items, meId, loading, onEdit, onDelete }) {
  const bottom = useRef(null)
  const [now, setNow] = useState(() => Date.now())
  const [editingId, setEditingId] = useState(0)
  const [draft, setDraft] = useState('')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    bottom.current?.scrollIntoView({ block: 'end' })
  }, [items.length])
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), TICK)
    return () => clearInterval(timer)
  }, [])
  const startEdit = (item) => {
    setEditingId(item.id)
    setDraft(item.body || '')
  }
  const cancelEdit = () => {
    setEditingId(0)
    setDraft('')
  }
  const save = async (item) => {
    setSaving(true)
    try {
      await onEdit(item, draft)
      cancelEdit()
    } finally {
      setSaving(false)
    }
  }
  if (loading && items.length === 0) {
    return <SkeletonLines lines={5} className="p-4" />
  }
  if (items.length === 0) {
    return (
      <EmptyState
        icon={MessagesSquare}
        title="No message yet"
        text="Write the first one below."
        className="m-3 border-0 py-10"
      />
    )
  }
  return (
    <div className="flex flex-col gap-4 px-3 py-4">
      {byDay(items).map((day) => (
        <section key={day.key}>
          <p className="glass-chip sticky top-2 z-10 mx-auto w-fit rounded-full px-3 py-1 text-2xs font-semibold text-muted">
            {dayLabel(day.items[0].created_at)}
          </p>
          <ul className="mt-3 flex flex-col gap-1">
            {day.items.map((item, index) => {
              const previous = day.items[index - 1]
              const mine = item.sender_id === meId
              const grouped =
                previous &&
                previous.sender_id === item.sender_id &&
                new Date(item.created_at).getTime() - new Date(previous.created_at).getTime() <
                  GROUP_WINDOW
              const editable =
                mine && !item.is_deleted && now - new Date(item.created_at).getTime() < EDIT_WINDOW
              const editing = editingId === item.id
              return (
                <li
                  key={item.id}
                  className={cx('flex gap-2', mine && 'flex-row-reverse', grouped ? 'mt-0.5' : 'mt-2')}
                >
                  <span className="w-8 shrink-0">
                    {!grouped && (
                      <Avatar src={item.photo_url} name={item.sender_name} size="sm" />
                    )}
                  </span>
                  <div className={cx('flex min-w-0 max-w-[78%] flex-col', mine && 'items-end')}>
                    {!grouped && !mine && (
                      <Link
                        to={paths.person(item.sender_id)}
                        className="mb-0.5 inline-flex items-center gap-1.5 px-1 text-2xs font-semibold text-muted hover:text-ink"
                      >
                        {item.sender_name}
                        <span className="text-faint">{shortRoleLabel(item.sender_role)}</span>
                      </Link>
                    )}
                    {item.is_deleted ? (
                      <p className="rounded-2xl border border-dashed border-line px-3.5 py-2 text-sm text-muted italic">
                        This message was deleted
                      </p>
                    ) : editing ? (
                      <div className="flex w-full flex-col gap-2">
                        <textarea
                          value={draft}
                          onChange={(event) => setDraft(event.target.value)}
                          maxLength={MAX_MESSAGE_BODY}
                          rows={3}
                          aria-label="Edit the message"
                          className="w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-100 focus:outline-none"
                        />
                        <div className="flex gap-2">
                          <Button size="sm" loading={saving} onClick={() => save(item)}>
                            Save
                          </Button>
                          <Button size="sm" variant="ghost" onClick={cancelEdit}>
                            Cancel
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <>
                        <div
                          className={cx(
                            'rounded-2xl px-3.5 py-2 text-sm',
                            mine
                              ? 'bg-brand-600 text-page'
                              : 'glass-card text-ink',
                            mine && !grouped && 'rounded-tr-md',
                            !mine && !grouped && 'rounded-tl-md',
                          )}
                        >
                          {item.body && (
                            <p className="wrap-long whitespace-pre-line">{item.body}</p>
                          )}
                          <AttachmentView
                            fileUrl={item.file_url}
                            fileName={item.file_name}
                            fileMime={item.file_mime}
                            fileSize={item.file_size}
                            linkUrl={item.link_url}
                            className={item.body ? 'mt-2' : ''}
                          />
                        </div>
                        <div className="mt-0.5 flex items-center gap-2 px-1">
                          <span className="text-2xs text-faint tabular-nums">
                            {clockOf(item.created_at)}
                          </span>
                          {item.edited_at && <span className="text-2xs text-faint">Edited</span>}
                          {editable && (
                            <>
                              <button
                                type="button"
                                onClick={() => startEdit(item)}
                                aria-label="Edit the message"
                                className="text-faint transition-colors hover:text-brand-600"
                              >
                                <Pencil size={13} />
                              </button>
                              <button
                                type="button"
                                onClick={() => onDelete(item)}
                                aria-label="Delete the message"
                                className="text-faint transition-colors hover:text-danger-600"
                              >
                                <Trash2 size={13} />
                              </button>
                            </>
                          )}
                        </div>
                      </>
                    )}
                  </div>
                </li>
              )
            })}
          </ul>
        </section>
      ))}
      <div ref={bottom} aria-hidden="true" />
    </div>
  )
}
