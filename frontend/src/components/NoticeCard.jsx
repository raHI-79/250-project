import AttachmentView from './AttachmentView'
import DateBlock from './DateBlock'
import { Avatar, Card, StatusBadge } from './ui'
import cx from '../lib/cx'
import { editedLabel, formatDateTime } from '../lib/time'

const SOURCE_LABELS = { admin: 'Department', teacher: 'Teacher', cr: 'CR' }

export default function NoticeCard({ item, highlighted = false, cardRef, title }) {
  return (
    <Card
      ref={cardRef}
      hover={false}
      data-notice={item.id}
      className={cx('animate-fade-in-up', highlighted && 'border-accent-300 bg-accent-50/60')}
    >
      <div className="flex gap-4">
        <DateBlock value={item.created_at} />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            {title || (
              <p className="font-display wrap-long text-lg font-bold text-ink">{item.title}</p>
            )}
            <StatusBadge
              status={item.source === 'admin' ? 'approved' : 'unavailable'}
              label={SOURCE_LABELS[item.source] || item.source}
            />
            {item.batch && (
              <StatusBadge
                status="unavailable"
                label={item.batch === 'ALL' ? 'All batches' : item.batch}
              />
            )}
            {item.section && <StatusBadge status="unavailable" label={`Section ${item.section}`} />}
            <span className="ml-auto text-xs whitespace-nowrap text-muted">
              {formatDateTime(item.created_at)}
              {item.edited_at ? ` · ${editedLabel(item.edited_at)}` : ''}
            </span>
          </div>
          {item.description && (
            <p className="wrap-long mt-2 text-sm leading-relaxed whitespace-pre-line text-ink-soft">
              {item.description}
            </p>
          )}
          <AttachmentView
            fileUrl={item.file_url}
            fileName={item.file_name}
            fileMime={item.file_mime}
            fileSize={item.file_size}
            linkUrl={item.link_url}
            className="mt-3"
          />
          {item.created_by_name && (
            <div className="mt-3 flex items-center gap-2 text-xs text-muted">
              <Avatar src={item.posted_by_photo_url} name={item.created_by_name} size="xs" />
              {item.created_by_name}
            </div>
          )}
        </div>
      </div>
    </Card>
  )
}
