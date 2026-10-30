import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  CalendarClock,
  CalendarDays,
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
  STATUS_CR_ASK,
  courseLine,
  isOnline,
  needsLink,
  placeOf,
  sectionShort,
} from '../../lib/schedule'
import { dayHeading, formatDate, formatTimeRange, today as todayDate } from '../../lib/time'
import cx from '../../lib/cx'
import { externalLinkProps } from '../../lib/native'

const TABS = [
  { value: TEACHER_TAB_NEXT_DAY, label: 'Upcoming days' },
  { value: TEACHER_TAB_FINAL, label: 'Final routine' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

const isExam = (item) => item.origin === ORIGIN_TEACHER_EXAM

export default function TeacherUpcomingPage() {
  const { user } = useAuth()
  const toast = useToast()
  const now = useNow(60000)
  const [tab, setTab] = useQueryTab(TAB_VALUES, TEACHER_TAB_NEXT_DAY)
  const upcoming = useResource('/api/teacher/upcoming')
  const final = useResource('/api/teacher/upcoming/final')
  const nextHighlight = useHighlight(Boolean(upcoming.data), 'item')
  const finalHighlight = useHighlight(Boolean(final.data), 'item')
  const [confirming, setConfirming] = useState(null)
  const [cancelling, setCancelling] = useState(null)
  const [dropping, setDropping] = useState(null)
  const [asking, setAsking] = useState(null)
  const [linking, setLinking] = useState(null)
  const [busy, setBusy] = useState(0)
  const upcomingDays = useMemo(() => upcoming.data?.days || [], [upcoming.data])
  const finalDays = useMemo(() => final.data?.days || [], [final.data])
  const items = useMemo(() => upcomingDays.flatMap((day) => day.items), [upcomingDays])
  const requests = items.filter((item) => item.status === STATUS_CR_ASK)
  const decisionDays = upcomingDays
    .map((day) => ({
      ...day,
      items: day.items.filter((item) => item.status !== STATUS_CR_ASK),
    }))
    .filter((day) => day.items.length > 0)
  const todayItems = finalDays.find((day) => day.date === todayDate())?.items || []
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
    await api.post(`/api/teacher/items/${cancelling.id}/cancel`, { reason })
    toast.success('Class cancelled')
    setCancelling(null)
    reload()
  }
  const cancelFromFinal = async (reason) => {
    const path = isExam(dropping)
      ? `/api/teacher/exams/${dropping.id}/cancel`
      : `/api/teacher/items/${dropping.id}/cancel`
    await api.post(path, { reason })
    toast.success(isExam(dropping) ? 'Exam cancelled' : 'Class cancelled')
    setDropping(null)
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
                  <a {...externalLinkProps(item.meeting_link)}>
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
            {upcoming.loading ? (
              <SkeletonCard />
            ) : upcoming.error ? (
              <Alert tone="error">{upcoming.error}</Alert>
            ) : decisionDays.length === 0 ? (
              <EmptyState
                icon={CalendarClock}
                title="Nothing to decide"
                text="Every class of the coming days is settled."
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
              <div className="flex flex-col gap-6">
                {decisionDays.map((day) => (
                  <section key={day.date}>
                    <p className="mb-2 text-sm font-semibold text-ink-soft">{dayHeading(day.date)}</p>
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
                  </section>
                ))}
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
                            <ItemBadges item={item} showOrigin />
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
                                onClick={() => setDropping(item)}
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
      <CancelModal item={cancelling} onClose={() => setCancelling(null)} onConfirm={cancelItem} />
      <CancelModal
        item={dropping}
        title={dropping && isExam(dropping) ? 'Cancel this exam?' : 'Cancel this class?'}
        confirmLabel={dropping && isExam(dropping) ? 'Cancel the exam' : 'Cancel the class'}
        onClose={() => setDropping(null)}
        onConfirm={cancelFromFinal}
      />
      <MessageModal item={asking} onClose={() => setAsking(null)} onConfirm={askMove} />
      <LinkModal item={linking} onClose={() => setLinking(null)} onConfirm={saveLink} />
    </PullToRefresh>
  )
}
