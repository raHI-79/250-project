import { Link } from 'react-router-dom'
import { Bookmark } from 'lucide-react'
import DateBlock from '../DateBlock'
import { Carousel, StatusBadge } from '../ui'
import { noticesFor } from '../../lib/paths'

const SOURCES = {
  admin: { label: 'Department', tone: 'brand' },
  teacher: { label: 'Teacher', tone: 'purple' },
  cr: { label: 'CR', tone: 'success' },
}

export default function NoticeCarousel({ notices, role }) {
  return (
    <Carousel label="Latest notices">
      {notices.map((notice) => {
        const source = SOURCES[notice.source] || { label: notice.source, tone: 'neutral' }
        return (
          <Link
            key={notice.id}
            to={noticesFor(role, { highlight: notice.id })}
            className="surface-card press flex w-[17rem] shrink-0 snap-start gap-3 p-4"
          >
            <DateBlock value={notice.created_at} />
            <div className="min-w-0 flex-1">
              <div className="flex items-start justify-between gap-2">
                <StatusBadge tone={source.tone} label={source.label} />
                <Bookmark size={15} className="shrink-0 text-faint" aria-hidden="true" />
              </div>
              <p className="mt-1.5 line-clamp-2 text-sm font-bold text-ink">{notice.title}</p>
              {notice.description && (
                <p className="mt-1 line-clamp-2 text-xs text-muted">{notice.description}</p>
              )}
              <p className="mt-1.5 wrap-long text-2xs text-faint">{notice.created_by_name}</p>
            </div>
          </Link>
        )
      })}
    </Carousel>
  )
}
