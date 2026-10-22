import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { CalendarClock } from 'lucide-react'
import DateChips from '../../components/DateChips'
import PublishedLine from '../../components/schedule/PublishedLine'
import { Link } from 'react-router-dom'
import { Alert, Button, Card, EmptyState, PageHeader, SkeletonCard } from '../../components/ui'
import { useResource } from '../../lib/hooks'
import paths from '../../lib/paths'
import { today } from '../../lib/time'

export default function UpcomingRoutinePage() {
  const [params, setParams] = useSearchParams()
  const [picked, setPicked] = useState(params.get('date') || today())
  useEffect(() => {
    const wanted = params.get('date')
    if (wanted && wanted !== picked) setPicked(wanted)
  }, [params, picked])
  const choose = (value) => {
    setPicked(value)
    setParams(
      (current) => {
        const copy = new URLSearchParams(current)
        copy.set('date', value)
        return copy
      },
      { replace: true },
    )
  }
  const { data, loading, error } = useResource('/api/students/upcoming-routine', { date: picked })
  const groups = data?.groups || []
  return (
    <>
      <PageHeader
        eyebrow="Published routine" title="Upcoming routine"
        subtitle="The routine your class representative published, one day at a time"
      />
      <DateChips value={picked} onChange={choose} label="Another day" />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : (
        <div className="flex flex-col gap-4">
          <p className="font-display text-lg font-semibold text-ink">
            {data.batch} · {data.date_label} ({data.weekday})
          </p>
          {groups.length === 0 ? (
            <EmptyState
              icon={CalendarClock}
              title="Nothing published"
              text="Your class representative has not published anything for this day yet."
              action={
                <Button as={Link} to={paths.student.fixedRoutine()} variant="secondary" size="sm">
                  Open the weekly routine
                </Button>
              }
            />
          ) : (
            groups.map((group) => (
              <Card key={group.key} hover={false} className="animate-fade-in-up">
                <p className="text-sm font-semibold text-ink-soft">{group.section_label}:</p>
                <ul className="mt-2 flex flex-col">
                  {group.items.map((item, index) => (
                    <PublishedLine
                      key={`${item.start_time}-${item.course_code}-${index}`}
                      item={item}
                      showBatch={group.drop_course}
                    />
                  ))}
                </ul>
              </Card>
            ))
          )}
        </div>
      )}
    </>
  )
}
