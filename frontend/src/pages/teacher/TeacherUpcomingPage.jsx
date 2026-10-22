import { useEffect, useState } from 'react'
import { CalendarClock, Clock, Link2 } from 'lucide-react'
import ItemBadges from '../../components/schedule/ItemBadges'
import {
  CancelModal,
  ConfirmSlotModal,
  LinkModal,
  MessageModal,
} from '../../components/schedule/ScheduleModals'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusTabs,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useHighlight, useQueryTab, useResource } from '../../lib/hooks'
import { TEACHER_TAB_FINAL, TEACHER_TAB_NEXT_DAY } from '../../lib/paths'
import {
  STATUS_CR_ASK,
  countdownText,
  courseLine,
  isOnline,
  placeOf,
  secondsUntilClock,
  sectionShort,
} from '../../lib/schedule'
import { formatDate, formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'

const TABS = [
  { value: TEACHER_TAB_NEXT_DAY, label: 'Next day' },
  { value: TEACHER_TAB_FINAL, label: 'Final routine' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

function WindowBanner({ open, start, end }) {
  const [seconds, setSeconds] = useState(() => secondsUntilClock(end))
  useEffect(() => {
    setSeconds(secondsUntilClock(end))
    const timer = window.setInterval(() => setSeconds(secondsUntilClock(end)), 1000)
    return () => window.clearInterval(timer)
  }, [end])
  if (!open) {
    return (
      <Alert tone="warning" title="The decision window is closed" className="mb-5">
        You can decide on tomorrow between {start} and {end}. Outside it the class representative
        decides, and you can still change a class you already confirmed until it starts.
      </Alert>
    )
  }
  return (
    <Alert tone="info" title={`The window is open until ${end}`} className="mb-5">
      <span className="inline-flex items-center gap-1.5">
        <Clock size={14} />
        {countdownText(seconds)} to confirm or move tomorrow's classes.
      </span>
    </Alert>
  )
}

export default function TeacherUpcomingPage() {
  const toast = useToast()
  const [tab, setTab] = useQueryTab(TAB_VALUES, TEACHER_TAB_NEXT_DAY)
  const nextDay = useResource(tab === TEACHER_TAB_NEXT_DAY ? '/api/teacher/upcoming/next-day' : null)
  const final = useResource(tab === TEACHER_TAB_FINAL ? '/api/teacher/upcoming/final' : null)
  const nextHighlight = useHighlight(Boolean(nextDay.data), 'item')
  const finalHighlight = useHighlight(Boolean(final.data), 'item')
  const [confirming, setConfirming] = useState(null)
  const [cancelling, setCancelling] = useState(null)
  const [asking, setAsking] = useState(null)
  const [linking, setLinking] = useState(null)
  const [busy, setBusy] = useState(0)
  const reload = () => {
    nextDay.reload()
    final.reload()
  }
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
  const finalItems = final.data?.items || []
  return (
    <>
      <PageHeader
        title="Upcoming routine"
        subtitle="Decide on tomorrow and keep an eye on what the batches published"
      />
      <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-5" />
      {tab === TEACHER_TAB_NEXT_DAY && (
        <>
          {nextDay.loading ? (
            <SkeletonCard />
          ) : nextDay.error ? (
            <Alert tone="error">{nextDay.error}</Alert>
          ) : (
            <>
              <WindowBanner
                open={nextDay.data.decision_open}
                start={nextDay.data.window_start}
                end={nextDay.data.window_end}
              />
              <p className="font-display mb-3 text-lg font-semibold text-brand-800">
                {formatDate(nextDay.data.date)} · {nextDay.data.weekday}
              </p>
              {items.length === 0 ? (
                <EmptyState
                  icon={CalendarClock}
                  title="Nothing tomorrow"
                  text="You have no class or exam on the next day."
                />
              ) : (
                <ul className="flex flex-col gap-3">
                  {items.map((item) => (
                    <li key={item.id} ref={nextHighlight.register(item.id)}>
                      <Card
                        hover={false}
                        className={cx(
                          'animate-fade-in-up',
                          nextHighlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
                        )}
                      >
                        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                          <span className="font-medium tabular-nums text-brand-600">
                            {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                          </span>
                          <span className="font-medium text-brand-900">{courseLine(item)}</span>
                          <span className="text-brand-500">
                            {item.batch} {sectionShort(item.section)}
                            {placeOf(item) ? ` · ${placeOf(item)}` : ''}
                          </span>
                          <ItemBadges item={item} />
                        </div>
                        {item.status === STATUS_CR_ASK && item.proposed && (
                          <Alert tone="info" title="The class representative proposed" className="mt-3">
                            {formatDate(item.proposed.item_date)}{' '}
                            {formatTimeRange(item.proposed.start_time, item.proposed.end_time)}
                            {item.proposed.room ? ` · ${item.proposed.room}` : ''}
                            {item.proposed.mode === 'online' ? ' · Online' : ''}
                          </Alert>
                        )}
                        {item.cancel_reason && (
                          <p className="mt-2 text-xs text-brand-500">
                            Reason: {item.cancel_reason}
                          </p>
                        )}
                        <div className="mt-3 flex flex-wrap gap-2">
                          {item.status === STATUS_CR_ASK ? (
                            <>
                              <Button
                                size="sm"
                                loading={busy === item.id}
                                onClick={() => decide(item, true)}
                              >
                                Accept
                              </Button>
                              <Button
                                size="sm"
                                variant="secondary"
                                loading={busy === item.id}
                                onClick={() => decide(item, false)}
                              >
                                Decline
                              </Button>
                            </>
                          ) : (
                            item.status !== 'cancelled' && (
                              <>
                                <Button size="sm" onClick={() => setConfirming(item)}>
                                  Confirm
                                </Button>
                                <Button
                                  size="sm"
                                  variant="secondary"
                                  onClick={() => setAsking(item)}
                                >
                                  Request reschedule
                                </Button>
                                <Button size="sm" variant="danger" onClick={() => setCancelling(item)}>
                                  Cancel
                                </Button>
                              </>
                            )
                          )}
                          {isOnline(item) && item.status !== 'cancelled' && (
                            <Button size="sm" variant="secondary" onClick={() => setLinking(item)}>
                              <Link2 size={14} />
                              Edit link
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
                      finalHighlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
                    )}
                  >
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                      <span className="w-28 shrink-0 text-sm font-medium text-brand-700">
                        {formatDate(item.item_date)}
                      </span>
                      <span className="font-medium tabular-nums text-brand-600">
                        {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                      </span>
                      <span className="font-medium text-brand-900">{courseLine(item)}</span>
                      <span className="text-brand-500">
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
                          Edit link
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
      <ConfirmSlotModal
        item={confirming}
        onClose={() => setConfirming(null)}
        onConfirm={confirmItem}
      />
      <CancelModal item={cancelling} onClose={() => setCancelling(null)} onConfirm={cancelItem} />
      <MessageModal item={asking} onClose={() => setAsking(null)} onConfirm={askMove} />
      <LinkModal item={linking} onClose={() => setLinking(null)} onConfirm={saveLink} />
    </>
  )
}
