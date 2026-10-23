import Avatar from '../ui/Avatar'
import CountBadge from '../ui/CountBadge'
import ListRow from '../ui/ListRow'
import { messagesFor } from '../../lib/paths'
import { relativeTime } from '../../lib/time'

export default function MessagePreview({ conversations, role }) {
  return (
    <div className="divide-y divide-line overflow-hidden rounded-card border border-line bg-surface">
      {conversations.map((conversation) => (
        <ListRow
          key={conversation.id}
          to={messagesFor(role, { c: conversation.id })}
          leading={<Avatar src={conversation.photo_url} name={conversation.title} size="md" />}
          title={conversation.title}
          subtitle={conversation.preview || 'No message yet'}
          value={conversation.last_message_at ? relativeTime(conversation.last_message_at) : ''}
          trailing={<CountBadge count={conversation.unread_count} />}
          chevron={false}
        />
      ))}
    </div>
  )
}
