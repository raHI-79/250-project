import { ClipboardList, Video } from 'lucide-react'
import { Alert, Button, Card, EmptyState, PageHeader, SkeletonCard, StatusBadge } from '../../components/ui'
import { useHighlight, useResource } from '../../lib/hooks'
import { courseLine, isOnline, sectionShort } from '../../lib/schedule'
import { formatDate, formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'

export default function ExamsPage() {
  const { data, loading, error } = useResource('/api/students/exams')
  const highlight = useHighlight(Boolean(data))
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Exams"
        subtitle="Every exam ahead with its syllabus; the time and the room arrive with the published routine"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title="No exams ahead"
          text="Nothing has been filed for your section yet."
        />
      ) : (
        <ul className="flex flex-col gap-3">
          {items.map((item) => (
            <li key={item.id} ref={highlight.register(item.id)}>
              <Card
                hover={false}
                className={cx(
                  'animate-fade-in-up',
                  highlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
                )}
              >
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                  <span className="font-display text-base font-semibold text-brand-800">
                    {formatDate(item.item_date)}
                  </span>
                  <span className="font-medium text-brand-900">{courseLine(item)}</span>
                  <span className="text-brand-500">
                    {item.batch} {sectionShort(item.section)}
                  </span>
                  {isOnline(item) && (
                    <span className="inline-flex items-center gap-1 rounded-full border border-brand-200 bg-brand-50 px-2.5 py-0.5 text-xs font-medium text-brand-700">
                      <Video size={12} />
                      Online
                    </span>
                  )}
                  {!item.published && <StatusBadge status="pending" label="Time to be announced" />}
                </div>
                <p className="mt-2 text-sm text-brand-700">
                  {item.start_time ? (
                    <>
                      <span className="font-medium tabular-nums">
                        {formatTimeRange(item.start_time, item.end_time)}
                      </span>
                      {isOnline(item) ? ' · Online' : item.room ? ` · Room ${item.room}` : ''}
                    </>
                  ) : (
                    'Time to be announced'
                  )}
                </p>
                {isOnline(item) && item.meeting_link && (
                  <a
                    href={item.meeting_link}
                    target="_blank"
                    rel="noreferrer"
                    className="mt-2 inline-block"
                  >
                    <Button size="sm">Join</Button>
                  </a>
                )}
                {item.syllabus && (
                  <p className="mt-2 text-sm text-brand-600">
                    <span className="font-medium text-brand-700">Syllabus: </span>
                    <span className="whitespace-pre-line">{item.syllabus}</span>
                  </p>
                )}
                {item.instructions && (
                  <p className="mt-1 text-sm text-brand-600 whitespace-pre-line">
                    {item.instructions}
                  </p>
                )}
              </Card>
            </li>
          ))}
        </ul>
      )}
    </>
  )
}
