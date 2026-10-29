import { Link } from 'react-router-dom'
import DateBlock from '../DateBlock'
import { Carousel } from '../ui'
import { courseCode, courseLine, placeOf } from '../../lib/schedule'
import { formatTimeRange } from '../../lib/time'

export default function ExamStrip({ exams, to }) {
  return (
    <Carousel label="Next exams">
      {exams.map((exam) => (
        <Link
          key={exam.id}
          to={`${to}${to.includes('?') ? '&' : '?'}highlight=${exam.id}`}
          className="glass-card press flex w-[15rem] shrink-0 snap-start gap-3 rounded-card p-4"
        >
          <DateBlock value={exam.item_date} />
          <div className="min-w-0 flex-1">
            <p className="line-clamp-2 text-sm font-semibold text-ink">{courseLine(exam)}</p>
            {courseCode(exam) && (
              <p className="mt-0.5 truncate text-2xs font-medium text-muted">{courseCode(exam)}</p>
            )}
            <p className="mt-1.5 truncate text-2xs text-faint tabular-nums">
              {[formatTimeRange(exam.start_time, exam.end_time), placeOf(exam)]
                .filter(Boolean)
                .join(' · ')}
            </p>
          </div>
        </Link>
      ))}
    </Carousel>
  )
}
