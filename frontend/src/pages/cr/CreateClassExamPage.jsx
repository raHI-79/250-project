import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import AvailabilityGrid from '../../components/schedule/AvailabilityGrid'
import DayPicker from '../../components/schedule/DayPicker'
import CreateTab from './tabs/CreateTab'
import FinalTab from './tabs/FinalTab'
import UpcomingTab from './tabs/UpcomingTab'
import { Alert, Card, Input, PageHeader, SkeletonCard, StatusTabs } from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useHighlight, useMoreDays, useQueryTab, useResource } from '../../lib/hooks'
import {
  CR_TAB_CREATE,
  CR_TAB_DEFAULT,
  CR_TAB_FINAL,
  CR_TAB_ORDER,
  CR_TAB_ROOMS,
  CR_TAB_UPCOMING,
} from '../../lib/paths'
import { defaultUpcomingDay, today } from '../../lib/time'

const LABELS = {
  [CR_TAB_UPCOMING]: 'Upcoming routine',
  [CR_TAB_FINAL]: 'Final routine',
  [CR_TAB_CREATE]: 'Create class / exam',
  [CR_TAB_ROOMS]: 'Room availability',
}

const TABS = CR_TAB_ORDER.map((value) => ({ value, label: LABELS[value] }))

export default function CreateClassExamPage() {
  const { user } = useAuth()
  const [params, setParams] = useSearchParams()
  const [tab, setTab] = useQueryTab(CR_TAB_ORDER, CR_TAB_DEFAULT)
  const [from, setFrom] = useState(params.get('date') || today())
  const [day, setDay] = useState(params.get('date') || defaultUpcomingDay())
  const [finalDay, setFinalDay] = useState(today())
  const [prefill, setPrefill] = useState(null)
  const courses = useResource('/api/cr/courses')
  const rooms = useResource(tab === CR_TAB_ROOMS ? '/api/cr/room-availability' : null, { from })
  const upcoming = useResource(tab === CR_TAB_UPCOMING ? '/api/cr/upcoming' : null, { date: day })
  const final = useResource(tab === CR_TAB_FINAL ? '/api/cr/final' : null, { date: finalDay })
  const horizon = useResource(tab === CR_TAB_UPCOMING ? '/api/cr/upcoming' : null, {
    from: today(),
    limit_days: 1,
  })
  const later = useMoreDays(
    tab === CR_TAB_UPCOMING ? '/api/cr/upcoming' : null,
    day,
    horizon.data?.horizon || '',
  )
  const upcomingHighlight = useHighlight(Boolean(upcoming.data), 'item')
  const finalHighlight = useHighlight(Boolean(final.data), 'item')
  const reloadAll = () => {
    upcoming.reload()
    final.reload()
    later.reset()
  }
  const pickSlot = (slot) => {
    setPrefill(slot)
    setParams(
      (current) => {
        const copy = new URLSearchParams(current)
        copy.set('tab', CR_TAB_CREATE)
        copy.set('date', slot.date)
        return copy
      },
      { replace: true },
    )
  }
  return (
    <>
      <PageHeader
        eyebrow="Class representative" title="Class & Exam hub"
        subtitle="Decide the coming days, keep the final routine, add classes and find a free room"
      />
      <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-5" />
      {tab === CR_TAB_UPCOMING && (
        <>
          <DayPicker value={day} onChange={setDay} toggle label="Jump to a date" />
          <UpcomingTab
            upcoming={upcoming}
            later={later}
            highlight={upcomingHighlight}
            onChanged={reloadAll}
          />
        </>
      )}
      {tab === CR_TAB_FINAL && (
        <>
          <DayPicker value={finalDay} onChange={setFinalDay} label="Jump to a date" />
          <FinalTab
            final={final}
            day={finalDay}
            highlight={finalHighlight}
            onPublished={reloadAll}
          />
        </>
      )}
      {tab === CR_TAB_CREATE && (
        <CreateTab
          courses={courses}
          ownSection={user?.section}
          prefill={prefill}
          onCreated={reloadAll}
        />
      )}
      {tab === CR_TAB_ROOMS && (
        <>
          <Card hover={false} className="mb-5">
            <Input
              label="Week starting on"
              type="date"
              value={from}
              min={today()}
              onChange={(event) => setFrom(event.target.value)}
              className="max-w-xs"
              hint="Seven days of every room a fixed routine uses"
            />
          </Card>
          {rooms.error && (
            <Alert tone="error" className="mb-5">
              {rooms.error}
            </Alert>
          )}
          {rooms.loading ? (
            <SkeletonCard />
          ) : (
            <AvailabilityGrid
              rooms={rooms.data?.rooms || []}
              periods={rooms.data?.periods || []}
              onPick={pickSlot}
            />
          )}
        </>
      )}
    </>
  )
}
