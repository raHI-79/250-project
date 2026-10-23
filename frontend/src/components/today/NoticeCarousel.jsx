import { Link } from 'react-router-dom'
import DateBlock from '../DateBlock'
import { Carousel, StatusBadge } from '../ui'
import { noticesFor } from '../../lib/paths'

const SOURCE_LABELS = { admin: 'Department', teacher: 'Teacher', cr: 'CR' }

export default function NoticeCarousel({ notices, role }) {
  return (
    <Carousel label="Latest notices">
      {notices.map((notice) => (
        <Link
          key={notice.id}
          to={noticesFor(role, { highlight: notice.id })}
          className="flex w-[17rem] shrink-0 snap-start gap-3 rounded-card border border-line bg-surface p-4 transition-colors active:bg-wash"
        >
          <DateBlock value={notice.created_at} />
          <div className="min-w-0 flex-1">
            <StatusBadge
              status="unavailable"
              label={SOURCE_LABELS[notice.source] || notice.source}
            />
            <p className="mt-1.5 line-clamp-2 text-sm font-semibold text-ink">{notice.title}</p>
            {notice.description && (
              <p className="mt-1 line-clamp-2 text-xs text-muted">{notice.description}</p>
            )}
            <p className="mt-1.5 truncate text-2xs text-faint">{notice.created_by_name}</p>
          </div>
        </Link>
      ))}
    </Carousel>
  )
}
