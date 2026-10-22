import { useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { MessagesSquare } from 'lucide-react'
import AttachmentView from '../AttachmentView'
import { Avatar, EmptyState, SkeletonLines, StatusBadge } from '../ui'
import cx from '../../lib/cx'
import { paths } from '../../lib/paths'
import { shortRoleLabel } from '../../lib/messages'
import { relativeTime } from '../../lib/time'

export default function MessageThread({ items, meId, loading }) {
  const bottom = useRef(null)
  useEffect(() => {
    bottom.current?.scrollIntoView({ block: 'end' })
  }, [items.length])
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
              </div>
              <div
                className={cx(
                  'mt-1 rounded-md px-3 py-2 text-sm',
                  mine
                    ? 'bg-brand-600 text-white'
                    : 'border border-line bg-surface text-ink',
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
            </div>
          </li>
        )
      })}
      <li ref={bottom} aria-hidden="true" />
    </ul>
  )
}
