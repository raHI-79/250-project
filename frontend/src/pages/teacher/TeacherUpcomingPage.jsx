import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  CalendarClock,
  CalendarDays,
  ChevronDown,
  ChevronUp,
  ClipboardList,
  Link2,
  Megaphone,
  MessageCircle,
  Video,
} from 'lucide-react'
import ItemBadges from '../../components/schedule/ItemBadges'
import {
  CancelModal,
  ConfirmSlotModal,
  LinkModal,
  MessageModal,
} from '../../components/schedule/ScheduleModals'
import QuickActions from '../../components/today/QuickActions'
import TeacherDecisionCard from '../../components/today/TeacherDecisionCard'
import TodayHero from '../../components/today/TodayHero'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  PullToRefresh,
  SkeletonCard,
  StatusTabs,
  useToast,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import api from '../../lib/api'
import { useHighlight, useNow, useQueryTab, useResource } from '../../lib/hooks'
import paths, { TEACHER_TAB_FINAL, TEACHER_TAB_NEXT_DAY } from '../../lib/paths'
import {
  ORIGIN_TEACHER_EXAM,
  STATUS_CANCELLED,
  STATUS_CONFIRMED,
  STATUS_RESCHEDULED,
  STATUS_SCHEDULED,
  courseLine,
  isOnline,
  needsLink,
  placeOf,
  sectionShort,
  teacherStatusLabel,
} from '../../lib/schedule'
import {
  dayHeading,
  formatDate,
  formatTimeRange,
  today as todayDate,
  tomorrow as tomorrowDate,
} from '../../lib/time'
import cx from '../../lib/cx'
import { externalLinkProps } from '../../lib/native'

const TABS = [
  { value: TEACHER_TAB_NEXT_DAY, label: 'Upcoming days' },
  { value: TEACHER_TAB_FINAL, label: 'Final routine' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

const SETTLED = [STATUS_CONFIRMED, STATUS_RESCHEDULED, STATUS_SCHEDULED]

const isExam = (item) => item.kind === 'exam'

const isOwnExam = (item) => item.origin === ORIGIN_TEACHER_EXAM

const settledToday = (item) =>
  item.status !== STATUS_CANCELLED && (SETTLED.includes(item.status) || item.published)

export default function TeacherUpcomingPage() {
  const { user } = useAuth()
  const toast = useToast()
  const now = useNow(60000)
  const [tab, setTab] = useQueryTab(TAB_VALUES, TEACHER_TAB_NEXT_DAY)
  const tomorrow = tomorrowDate()
  const upcoming = useResource('/api/teacher/upcoming', { from: tomorrow })
  const final = useResource('/api/teacher/upcoming/final')
  const nextHighlight = useHighlight(Boolean(upcoming.data), 'item')
  const finalHighlight = useHighlight(Boolean(final.data), 'item')
  const [confirming, setConfirming] = useState(null)
  const [cancelling, setCancelling] = useState(null)
  const [asking, setAsking] = useState(null)
  const [linking, setLinking] = useState(null)
  const [showLater, setShowLater] = useState(false)
  const [busy, setBusy] = useState(0)
  const upcomingDays = useMemo(() => upcoming.data?.days || [], [upcoming.data])
  const finalDays = useMemo(() => final.data?.days || [], [final.data])
  const tomorrowItems = upcomingDays.find((day) => day.date === tomorrow)?.items || []
  const laterDays = upcomingDays.filter((day) => day.date > tomorrow)
  const todayItems = (finalDays.find((day) => day.date === todayDate())?.items || []).filter(
    settledToday,
  )
  const reload = () => Promise.all([upcoming.reload(), final.reload()])
  const act = async (id, call, message) => {
    setBusy(id)
    try {
      await call()
      toast.success(message)
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const confirmItem = async (slot) => {
    await api.post(`/api/teacher/items/${confirming.id}/confirm`, {
      mode: slot.mode,
      room: slot.room,
      meeting_link: slot.meeting_link,
    })
    toast.success('Class confirmed')
    setConfirming(null)
    reload()
  }
  const cancelItem = async (reason) => {
    const path = isOwnExam(cancelling)
      ? `/api/teacher/exams/${cancelling.id}/cancel`
      : `/api/teacher/items/${cancelling.id}/cancel`
    await api.post(path, { reason })
    toast.success(isExam(cancelling) ? 'Exam cancelled' : 'Class cancelled')
    setCancelling(null)
    reload()
  }
  const askMove = async (message) => {
    await api.post(`/api/teacher/items/${asking.id}/request-reschedule`, { message })
    toast.success('The class representative was asked for another slot')
    setAsking(null)
    reload()
  }
  const saveLink = async (link) => {
    await api.patch(`/api/schedule/items/${linking.id}/link`, { meeting_link: link })
    toast.success('Meeting link updated')
    setLinking(null)
    reload()
  }
  const decide = (item, accept) =>
    act(
      item.id,
      () =>
        api.post(
          `/api/teacher/items/${item.id}/reschedule-request/${accept ? 'accept' : 'decline'}`,
        ),
      accept ? 'The new slot is in place' : 'The original slot stays',
    )
  const actions = [
    { label: 'Create exam', icon: ClipboardList, to: paths.teacher.createExam() },
    { label: 'Post notice', icon: Megaphone, to: paths.teacher.notices() },
    { label: 'Messages', icon: MessageCircle, to: paths.teacher.messages() },
    { label: 'Weekly routine', icon: CalendarDays, to: paths.teacher.fixedRoutine() },
  ]
  const cardProps = {
    onConfirm: setConfirming,
    onCancel: setCancelling,
    onAsk: setAsking,
    onLink: setLinking,
    onDecide: decide,
  }
  const dayList = (day) => (
    <ul className="flex flex-col gap-3">
      {day.items.map((item) => (
        <li key={item.id} ref={nextHighlight.register(item.id)}>
          <TeacherDecisionCard
            item={item}
            busy={busy === item.id}
            highlighted={nextHighlight.highlighted(item.id)}
            {...cardProps}
          />
        </li>
      ))}
    </ul>
  )
  return (
    <PullToRefresh onRefresh={reload}>
      <TodayHero
        name={user?.name || user?.identifier}
        meta={user?.designation || 'Teacher'}
        now={now}
      />
      <QuickActions actions={actions} className="mt-6 grid-cols-4" />
      <section className="mt-7">
        <h2 className="font-display mb-3 text-section font-bold text-ink">
          {dayHeading(todayDate())}
        </h2>
        {final.loading ? (
          <SkeletonCard />
        ) : todayItems.length === 0 ? (
          <EmptyState
            icon={CalendarClock}
            title="No confirmed classes today"
            text="Classes and exams you confirmed for today show up here."
          />
        ) : (
          <Card hover={false} flush className="divide-y divide-line">
            {todayItems.map((item) => (
              <div key={item.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                <span className="w-24 shrink-0 text-xs font-semibold text-brand-600 tabular-nums">
                  {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-semibold text-ink">
                    {courseLine(item)}
                  </span>
                  <span className="block truncate text-xs text-muted">
                    {item.batch} {sectionShort(item.section)}
                    {placeOf(item) ? ` · ${placeOf(item)}` : ''}
                  </span>
                </span>
                {isOnline(item) && item.meeting_link && (
                  <a {...externalLinkProps(item.meeting_link)}>
                    <Button as="span" size="sm" variant="secondary">
                      <Video size={14} />
                      Join
                    </Button>
                  </a>
                )}
                <Button size="sm" variant="danger" onClick={() => setCancelling(item)}>
                  Cancel
                </Button>
              </div>
            ))}
          </Card>
        )}
      </section>
      <section className="mt-7">
        <h2 className="font-display mb-3 text-section font-bold text-ink">Upcoming</h2>
        <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-4" />
        {tab === TEACHER_TAB_NEXT_DAY && (
          <>
            {upcoming.loading ? (
              <SkeletonCard />
            ) : upcoming.error ? (
              <Alert tone="error">{upcoming.error}</Alert>
            ) : (
              <div className="flex flex-col gap-6">
                <section>
                  <p className="mb-2 text-sm font-semibold text-ink-soft">{dayHeading(tomorrow)}</p>
                  {tomorrowItems.length === 0 ? (
                    <EmptyState
                      icon={CalendarClock}
                      title="No classes tomorrow"
                      text="Nothing of yours is on the routine for the next day."
                      action={
                        <Button
                          as={Link}
                          to={paths.teacher.fixedRoutine()}
                          variant="secondary"
                          size="sm"
                        >
                          Open your weekly routine
                        </Button>
                      }
                    />
                  ) : (
                    dayList({ items: tomorrowItems })
                  )}
                </section>
                {laterDays.length > 0 && (
                  <>
                    <Button
                      variant="secondary"
                      size="sm"
                      className="self-start"
                      onClick={() => setShowLater((current) => !current)}
                    >
                      {showLater ? <ChevronUp size={15} /> : <ChevronDown size={15} />}
                      {showLater ? 'Hide later days' : 'Show later days'}
                    </Button>
                    {showLater &&
                      laterDays.map((day) => (
                        <section key={day.date}>
                          <p className="mb-2 text-sm font-semibold text-ink-soft">
                            {dayHeading(day.date)}
                          </p>
                          {dayList(day)}
                        </section>
                      ))}
                  </>
                )}
              </div>
            )}
          </>
        )}
        {tab === TEACHER_TAB_FINAL && (
          <>
            {final.loading ? (
              <SkeletonCard />
            ) : final.error ? (
              <Alert tone="error">{final.error}</Alert>
            ) : finalDays.length === 0 ? (
              <EmptyState
                icon={CalendarClock}
                title="Nothing in the final routine"
                text="Classes you confirmed and exams you filed show up here from today onward."
              />
            ) : (
              <div className="flex flex-col gap-4">
                {finalDays.map((day) => (
                  <Card key={day.date} hover={false} className="animate-fade-in-up">
                    <p className="font-display text-lg font-bold text-ink">{dayHeading(day.date)}</p>
                    <ul className="mt-2 flex flex-col gap-3">
                      {day.items.map((item) => (
                        <li
                          key={item.id}
                          ref={finalHighlight.register(item.id)}
                          className={cx(
                            'rounded-xl border border-line p-3 transition-colors',
                            finalHighlight.highlighted(item.id) &&
                              'border-brand-300 bg-brand-50/60',
                          )}
                        >
                          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                            <span className="w-28 shrink-0 text-sm font-medium text-ink-soft">
                              {formatDate(item.item_date)}
                            </span>
                            <span className="font-medium text-brand-600 tabular-nums">
                              {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                            </span>
                            <span className="font-medium text-ink">{courseLine(item)}</span>
                            <span className="text-muted">
                              {item.batch} {sectionShort(item.section)}
                              {placeOf(item) ? ` · ${placeOf(item)}` : ''}
                            </span>
                            <ItemBadges
                              item={item}
                              showOrigin
                              statusText={teacherStatusLabel(item)}
                            />
                          </div>
                          {item.cancel_reason && (
                            <p className="mt-1 text-xs text-muted">Reason: {item.cancel_reason}</p>
                          )}
                          {item.status !== STATUS_CANCELLED && (
                            <div className="mt-2.5 flex flex-wrap gap-2">
                              {isOnline(item) && (
                                <Button
                                  size="sm"
                                  variant="secondary"
                                  onClick={() => setLinking(item)}
                                >
                                  <Link2 size={14} />
                                  {needsLink(item) ? 'Add link' : 'Edit link'}
                                </Button>
                              )}
                              <Button
                                size="sm"
                                variant="danger"
                                onClick={() => setCancelling(item)}
                              >
                                {isExam(item) ? 'Cancel exam' : 'Cancel class'}
                              </Button>
                            </div>
                          )}
                        </li>
                      ))}
                    </ul>
                  </Card>
                ))}
              </div>
            )}
          </>
        )}
      </section>
      <ConfirmSlotModal
        item={confirming}
        onClose={() => setConfirming(null)}
        onConfirm={confirmItem}
      />
      <CancelModal
        item={cancelling}
        title={cancelling && isExam(cancelling) ? 'Cancel this exam?' : 'Cancel this class?'}
        confirmLabel={cancelling && isExam(cancelling) ? 'Cancel the exam' : 'Cancel the class'}
        onClose={() => setCancelling(null)}
        onConfirm={cancelItem}
      />
      <MessageModal item={asking} onClose={() => setAsking(null)} onConfirm={askMove} />
      <LinkModal item={linking} onClose={() => setLinking(null)} onConfirm={saveLink} />
    </PullToRefresh>
  )
}
