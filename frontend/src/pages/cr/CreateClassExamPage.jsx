import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import AvailabilityGrid from '../../components/schedule/AvailabilityGrid'
import CreateTab from './tabs/CreateTab'
import FinalTab from './tabs/FinalTab'
import UpcomingTab from './tabs/UpcomingTab'
import { Alert, Card, Input, PageHeader, SkeletonCard, StatusTabs } from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useHighlight, useQueryTab, useResource } from '../../lib/hooks'
import {
  CR_TAB_CREATE,
  CR_TAB_FINAL,
  CR_TAB_ROOMS,
  CR_TAB_UPCOMING,
} from '../../lib/paths'
import { today } from '../../lib/time'

const TABS = [
  { value: CR_TAB_CREATE, label: 'Create' },
  { value: CR_TAB_ROOMS, label: 'Room availability' },
  { value: CR_TAB_UPCOMING, label: 'Upcoming routine' },
  { value: CR_TAB_FINAL, label: 'Final routine' },
]

const TAB_VALUES = TABS.map((entry) => entry.value)

export default function CreateClassExamPage() {
  const { user } = useAuth()
  const [params, setParams] = useSearchParams()
  const [tab, setTab] = useQueryTab(TAB_VALUES, CR_TAB_CREATE)
  const [from, setFrom] = useState(params.get('date') || today())
  const [prefill, setPrefill] = useState(null)
  const courses = useResource('/api/cr/courses')
  const rooms = useResource(tab === CR_TAB_ROOMS ? '/api/cr/room-availability' : null, { from })
  const upcoming = useResource(tab === CR_TAB_UPCOMING ? '/api/cr/upcoming' : null, { from })
  const final = useResource(tab === CR_TAB_FINAL ? '/api/cr/final' : null)
  const upcomingHighlight = useHighlight(Boolean(upcoming.data), 'item')
  const finalHighlight = useHighlight(Boolean(final.data), 'item')
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
        eyebrow="Class representative" title="Create Class & Exam"
        subtitle="Add classes and exams, find a free room, decide what your batch sees"
      />
      <StatusTabs tabs={TABS} value={tab} onChange={setTab} className="mb-5" />
      {tab === CR_TAB_CREATE && (
        <CreateTab
          courses={courses}
          ownSection={user?.section}
          prefill={prefill}
          onCreated={() => {
            upcoming.reload()
            final.reload()
          }}
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
      {tab === CR_TAB_UPCOMING && (
        <>
          <Card hover={false} className="mb-5">
            <Input
              label="From"
              type="date"
              value={from}
              onChange={(event) => setFrom(event.target.value)}
              className="max-w-xs"
            />
          </Card>
          <UpcomingTab upcoming={upcoming} highlight={upcomingHighlight} />
        </>
      )}
      {tab === CR_TAB_FINAL && (
        <FinalTab final={final} highlight={finalHighlight} onPublished={() => upcoming.reload()} />
      )}
    </>
  )
}
