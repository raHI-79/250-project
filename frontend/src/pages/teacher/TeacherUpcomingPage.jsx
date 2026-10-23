import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  CalendarClock,
  CalendarDays,
  ClipboardList,
  Clock,
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
  STATUS_AWAITING,
  STATUS_CR_ASK,
  STATUS_NO_RESPONSE,
  countdownText,
  courseLine,
  isOnline,
  needsLink,
  placeOf,
  secondsUntilClock,
  sectionShort,
} from '../../lib/schedule'
import { formatClock, formatDate, formatTimeRange, today as todayDate } from '../../lib/time'
import cx from '../../lib/cx'

const TABS = [
  { value: TEACHER_TAB_NEXT_DAY, label: 'Tomorrow' },
  { value: TEACHER_TAB_FINAL, label: 'Final routine' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

const lockedHint = 'The decision window is closed for a first decision on this class.'

function WindowLine({ state, start, end }) {
  const [seconds, setSeconds] = useState(() => secondsUntilClock(end))
  useEffect(() => {
    setSeconds(secondsUntilClock(end))
    const timer = window.setInterval(() => setSeconds(secondsUntilClock(end)), 1000)
    return () => window.clearInterval(timer)
  }, [end])
  if (state === 'before') {
    return (
      <Alert tone="info">
        <span className="inline-flex items-center gap-1.5">
          <Clock size={14} />
          Opens at {formatClock(start)} for tomorrow&apos;s classes
        </span>
      </Alert>
    )
  }
  if (state === 'closed') {
    return (
      <Alert tone="warning">
        Closed for tomorrow — undecided classes are marked No response
      </Alert>
    )
  }
  return (
    <Alert tone="info">
      <span className="inline-flex items-center gap-1.5">
        <Clock size={14} />
        Open until {formatClock(end)} · {countdownText(seconds)}
      </span>
    </Alert>
  )
}

export default function TeacherUpcomingPage() {
  const { user } = useAuth()
  const toast = useToast()
  const now = useNow(60000)
  const [tab, setTab] = useQueryTab(TAB_VALUES, TEACHER_TAB_NEXT_DAY)
  const nextDay = useResource('/api/teacher/upcoming/next-day')
  const final = useResource('/api/teacher/upcoming/final')
  const nextHighlight = useHighlight(Boolean(nextDay.data), 'item')
  const finalHighlight = useHighlight(Boolean(final.data), 'item')
  const [confirming, setConfirming] = useState(null)
  const [cancelling, setCancelling] = useState(null)
  const [asking, setAsking] = useState(null)
  const [linking, setLinking] = useState(null)
  const [busy, setBusy] = useState(0)
  const reload = () => Promise.all([nextDay.reload(), final.reload()])
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
    await api.post(`/api/teacher/items/${cancelling.id}/cancel`, { reason })
    toast.success('Class cancelled')
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
  const items = nextDay.data?.items || []
  const windowOpen = Boolean(nextDay.data?.decision_open)
  const locked = (item) =>
    !windowOpen && (item.status === STATUS_AWAITING || item.status === STATUS_NO_RESPONSE)
  const finalItems = final.data?.items || []
  const requests = items.filter((item) => item.status === STATUS_CR_ASK)
  const decisions = items.filter((item) => item.status !== STATUS_CR_ASK)
  const todayItems = finalItems.filter((item) => item.item_date === todayDate())
  const actions = [
    { label: 'Create exam', icon: ClipboardList, to: paths.teacher.createExam() },
    { label: 'Post notice', icon: Megaphone, to: paths.teacher.notices() },
    { label: 'Messages', icon: MessageCircle, to: paths.teacher.messages() },
    { label: 'Weekly routine', icon: CalendarDays, to: paths.teacher.fixedRoutine() },
  ]
  const cardProps = {
    lockedHint,
    onConfirm: setConfirming,
    onCancel: setCancelling,
    onAsk: setAsking,
    onLink: setLinking,
    onDecide: decide,
  }
  return (
    <PullToRefresh onRefresh={reload}>
      <TodayHero
        name={user?.name || user?.identifier}
        meta={user?.designation || 'Teacher'}
        now={now}
      />
      <QuickActions actions={actions} className="mt-6 grid-cols-4" />
      {todayItems.length > 0 && (
        <section className="mt-7">
          <h2 className="font-display mb-3 text-section font-bold text-ink">Today</h2>
          <Card hover={false} flush className="divide-y divide-line">
            {todayItems.map((item) => (
              <div key={item.id} className="flex items-center gap-3 px-4 py-3">
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
                  <a href={item.meeting_link} target="_blank" rel="noreferrer">
                    <Button as="span" size="sm" variant="secondary">
                      <Video size={14} />
                      Join
                    </Button>
                  </a>
                )}
              </div>
            ))}
          </Card>
        </section>
      )}
      {requests.length > 0 && (
        <section className="mt-7">
          <h2 className="font-display mb-3 text-section font-bold text-ink">
            Reschedule requests
          </h2>
          <ul className="flex flex-col gap-3">
            {requests.map((item) => (
              <li key={item.id} ref={nextHighlight.register(item.id)}>
                <TeacherDecisionCard
                  item={item}
                  busy={busy === item.id}
                  locked={false}
                  highlighted={nextHighlight.highlighted(item.id)}
                  {...cardProps}
                />
              </li>
            ))}
          </ul>
        </section>
      )}
      <section className="mt-7">
        <h2 className="font-display mb-3 text-section font-bold text-ink">Upcoming</h2>
        <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-4" />
        {tab === TEACHER_TAB_NEXT_DAY && (
          <>
            {nextDay.loading ? (
              <SkeletonCard />
            ) : nextDay.error ? (
              <Alert tone="error">{nextDay.error}</Alert>
            ) : (
              <>
                <WindowLine
                  state={nextDay.data.decision_state}
                  start={nextDay.data.window_start}
                  end={nextDay.data.window_end}
                />
                <p className="mt-4 mb-3 text-sm font-semibold text-ink-soft">
                  {formatDate(nextDay.data.date)} · {nextDay.data.weekday}
                </p>
                {items.length === 0 ? (
                  <EmptyState
                    icon={CalendarClock}
                    title="Nothing tomorrow"
                    text="You have no class or exam on the next day."
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
                  <ul className="flex flex-col gap-3">
                    {decisions.map((item) => (
                      <li key={item.id} ref={nextHighlight.register(item.id)}>
                        <TeacherDecisionCard
                          item={item}
                          busy={busy === item.id}
                          locked={locked(item)}
                          highlighted={nextHighlight.highlighted(item.id)}
                          {...cardProps}
                        />
                      </li>
                    ))}
                  </ul>
                )}
              </>
            )}
          </>
        )}
        {tab === TEACHER_TAB_FINAL && (
          <>
            {final.loading ? (
              <SkeletonCard />
            ) : final.error ? (
              <Alert tone="error">{final.error}</Alert>
            ) : finalItems.length === 0 ? (
              <EmptyState
                icon={CalendarClock}
                title="Nothing in the final routine"
                text="Classes you confirmed and exams you filed show up here from today onward."
              />
            ) : (
              <ul className="flex flex-col gap-3">
                {finalItems.map((item) => (
                  <li key={item.id} ref={finalHighlight.register(item.id)}>
                    <Card
                      hover={false}
                      className={cx(
                        'animate-fade-in-up',
                        finalHighlight.highlighted(item.id) && 'border-brand-300 bg-brand-50/60',
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
                        <ItemBadges item={item} showOrigin />
                        {isOnline(item) && item.status !== 'cancelled' && (
                          <Button
                            size="sm"
                            variant="secondary"
                            className="ml-auto"
                            onClick={() => setLinking(item)}
                          >
                            <Link2 size={14} />
                            {needsLink(item) ? 'Add link' : 'Edit link'}
                          </Button>
                        )}
                      </div>
                    </Card>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </section>
      <ConfirmSlotModal
        item={confirming}
        onClose={() => setConfirming(null)}
        onConfirm={confirmItem}
      />
      <CancelModal item={cancelling} onClose={() => setCancelling(null)} onConfirm={cancelItem} />
      <MessageModal item={asking} onClose={() => setAsking(null)} onConfirm={askMove} />
      <LinkModal item={linking} onClose={() => setLinking(null)} onConfirm={saveLink} />
    </PullToRefresh>
  )
}
