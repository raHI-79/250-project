import { ClipboardList, Video } from 'lucide-react'
import { Link } from 'react-router-dom'
import DateBlock from '../../components/DateBlock'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  PullToRefresh,
} from '../../components/ui'
import { useHighlight, useResource } from '../../lib/hooks'
import paths from '../../lib/paths'
import CourseName from '../../components/schedule/CourseName'
import { isOnline, sectionShort } from '../../lib/schedule'
import { formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'
import { BADGE_EXAMS, useSectionSeen } from '../../lib/badges'

export default function ExamsPage() {
  useSectionSeen(BADGE_EXAMS)
  const { data, loading, error, reload } = useResource('/api/students/exams')
  const highlight = useHighlight(Boolean(data))
  const items = data?.items || []
  return (
    <PullToRefresh onRefresh={reload}>
      <PageHeader
        eyebrow="Examinations"
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
          action={
            <Button as={Link} to={paths.student.upcomingRoutine()} variant="secondary" size="sm">
              Read the upcoming routine
            </Button>
          }
        />
      ) : (
        <ul className="flex flex-col gap-3">
          {items.map((item) => (
            <li key={item.id} ref={highlight.register(item.id)}>
              <Card
                hover={false}
                className={cx(
                  'animate-fade-in-up',
                  highlight.highlighted(item.id) && 'animate-flash border-gold-300',
                )}
              >
                <div className="flex gap-4">
                  <DateBlock value={item.item_date} />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
                      <span className="font-display text-lg font-bold text-ink">
                        <CourseName item={item} />
                      </span>
                      <StatusBadge
                        status="unavailable"
                        label={`${item.batch} ${sectionShort(item.section)}`}
                      />
                      {isOnline(item) && <StatusBadge status="gold" label="Online" />}
                      {!item.published && (
                        <StatusBadge status="pending" label="Time to be announced" />
                      )}
                    </div>
                    <p className="font-numeral mt-1.5 text-sm text-ink-soft">
                      {item.start_time ? (
                        <>
                          {formatTimeRange(item.start_time, item.end_time)}
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
                        <Button size="sm" variant="secondary">
                          <Video size={13} />
                          Join
                        </Button>
                      </a>
                    )}
                    {item.syllabus && (
                      <div className="mt-3 border-l-[3px] border-gold-500 bg-gold-50 px-3 py-2">
                        <p className="label-caps text-gold-700">Syllabus</p>
                        <p className="mt-1 text-sm leading-relaxed whitespace-pre-line text-gold-800">
                          {item.syllabus}
                        </p>
                      </div>
                    )}
                    {item.instructions && (
                      <p className="mt-2 text-sm leading-relaxed whitespace-pre-line text-muted">
                        {item.instructions}
                      </p>
                    )}
                  </div>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </PullToRefresh>
  )
}
