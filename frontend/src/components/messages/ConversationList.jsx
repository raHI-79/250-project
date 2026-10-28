import { MessageSquare, Users } from 'lucide-react'
import { Avatar, CountBadge, EmptyState, SkeletonRows } from '../ui'
import cx from '../../lib/cx'
import { relativeTime } from '../../lib/time'

export default function ConversationList({ items, activeId, onSelect, search, loading, header }) {
  const text = search.trim().toLowerCase()
  const shown = text
    ? items.filter((item) => `${item.title} ${item.preview}`.toLowerCase().includes(text))
    : items
  return (
    <div className="glass-card flex h-full min-h-0 flex-col overflow-hidden rounded-card">
      {header}
      <div className="min-h-0 flex-1 overflow-y-auto">
        {loading ? (
          <SkeletonRows rows={6} className="border-0" />
        ) : shown.length === 0 ? (
          <EmptyState
            icon={MessageSquare}
            title="No conversations"
            text="Search for a person above to start one."
            className="m-3 border-0 py-8"
          />
        ) : (
          <ul>
            {shown.map((item) => (
              <li key={item.id}>
                <button
                  type="button"
                  onClick={() => onSelect(item)}
                  className={cx(
                    'flex min-h-16 w-full items-center gap-3 border-b border-line px-3 text-left transition-colors last:border-b-0',
                    item.id === activeId ? 'bg-brand-50' : 'hover:bg-wash',
                  )}
                >
                  {item.kind === 'group' ? (
                    <span className="inline-flex size-10 shrink-0 items-center justify-center rounded-full border border-line bg-wash text-brand-600">
                      <Users size={18} />
                    </span>
                  ) : (
                    <Avatar src={item.photo_url} name={item.title} />
                  )}
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center gap-2">
                      <span className="truncate text-sm font-semibold text-ink">
                        {item.title}
                      </span>
                      <span className="ml-auto shrink-0 text-xs whitespace-nowrap text-faint">
                        {relativeTime(item.last_message_at)}
                      </span>
                    </span>
                    <span className="mt-0.5 flex items-center gap-2">
                      <span className="truncate text-xs text-muted">
                        {item.preview || 'No message yet'}
                      </span>
                      <CountBadge count={item.unread_count} className="ml-auto shrink-0" />
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
