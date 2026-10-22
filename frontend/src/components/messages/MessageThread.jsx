import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { MessagesSquare, Pencil, Trash2 } from 'lucide-react'
import AttachmentView from '../AttachmentView'
import { Avatar, Button, EmptyState, SkeletonLines, StatusBadge } from '../ui'
import cx from '../../lib/cx'
import { paths } from '../../lib/paths'
import { EDIT_WINDOW, MAX_MESSAGE_BODY, shortRoleLabel } from '../../lib/messages'
import { relativeTime } from '../../lib/time'

const TICK = 10000

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
    <ul className="flex flex-col gap-4 p-4">
      {items.map((item) => {
        const mine = item.sender_id === meId
        const editable =
          mine && !item.is_deleted && now - new Date(item.created_at).getTime() < EDIT_WINDOW
        const editing = editingId === item.id
        return (
          <li key={item.id} className={cx('flex gap-2.5', mine && 'flex-row-reverse')}>
            <Avatar src={item.photo_url} name={item.sender_name} size="sm" />
            <div className={cx('flex min-w-0 max-w-[85%] flex-col', mine && 'items-end')}>
              <div className="flex flex-wrap items-center gap-2 text-xs">
                {mine ? (
                  <span className="font-semibold text-ink">You</span>
                ) : (
                  <Link
                    to={paths.person(item.sender_id)}
                    className="font-semibold text-ink hover:underline"
                  >
                    {item.sender_name}
                  </Link>
                )}
                <StatusBadge status="unavailable" label={shortRoleLabel(item.sender_role)} />
                <span className="text-faint">{relativeTime(item.created_at)}</span>
                {item.edited_at && !item.is_deleted && <span className="text-faint">Edited</span>}
              </div>
              {item.is_deleted ? (
                <p className="mt-1 rounded-md border border-dashed border-line px-3 py-2 text-sm text-muted italic">
                  This message was deleted
                </p>
              ) : editing ? (
                <div className="mt-1 flex w-full flex-col gap-2">
                  <textarea
                    value={draft}
                    onChange={(event) => setDraft(event.target.value)}
                    maxLength={MAX_MESSAGE_BODY}
                    rows={3}
                    aria-label="Edit the message"
                    className="w-full rounded-md border border-line-strong bg-surface px-3 py-2 text-sm text-ink focus:border-brand-500 focus:ring-2 focus:ring-brand-100 focus:outline-none"
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
                      'mt-1 rounded-md px-3 py-2 text-sm',
                      mine ? 'bg-brand-600 text-white' : 'border border-line bg-surface text-ink',
                    )}
                  >
                    {item.body && <p className="break-words whitespace-pre-line">{item.body}</p>}
                    <AttachmentView
                      fileUrl={item.file_url}
                      fileName={item.file_name}
                      fileMime={item.file_mime}
                      fileSize={item.file_size}
                      linkUrl={item.link_url}
                      className={item.body ? 'mt-2' : ''}
                    />
                  </div>
                  {editable && (
                    <div className="mt-1 flex gap-1">
                      <Button size="sm" variant="ghost" onClick={() => startEdit(item)}>
                        <Pencil size={14} />
                        Edit
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => onDelete(item)}>
                        <Trash2 size={14} />
                        Delete
                      </Button>
                    </div>
                  )}
                </>
              )}
            </div>
          </li>
        )
      })}
      <li ref={bottom} aria-hidden="true" />
    </ul>
  )
}
