import { useState } from 'react'
import { Alert, Card, PageHeader, Select, SkeletonCard, StatusTabs } from '../../components/ui'
import RoutineWeekGrid from '../../components/RoutineWeekGrid'
import { useAuth } from '../../context/auth'
import { usePeriods, useResource } from '../../lib/hooks'
import { SECTIONS, semesterLabel, semesterOptions } from '../../lib/options'

const SECTION_TABS = SECTIONS.map((section) => ({ value: section, label: `Section ${section}` }))

const batchOptions = (batches, own) => {
  const all = batches || []
  if (!own) return all
  return [own, ...all.filter((entry) => entry !== own)]
}

export default function FixedRoutinePage() {
  const { user } = useAuth()
  const [batch, setBatch] = useState(user?.batch || '')
  const [semester, setSemester] = useState('')
  const [tab, setTab] = useState(user?.section || SECTIONS[0])
  const periods = usePeriods()
  const batches = useResource('/api/lookups/batches')
  const query = {}
  if (batch) query.batch = batch
  if (semester) query.semester = semester
  const { data, loading, error } = useResource('/api/students/routine', query)
  const items = data?.items || []
  const running = data?.running_semester || ''
  const shown = semester || data?.semester || ''
  return (
    <>
      <PageHeader
        eyebrow="Weekly routine"
        title="Fixed Routine"
        subtitle={
          shown
            ? `${batch || user?.batch || ''} · ${semesterLabel(shown)}`
            : 'The weekly routine of a batch, section by section'
        }
      />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            {batchOptions(batches.data?.items, user?.batch).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <Select
            label="Semester"
            value={shown}
            onChange={(event) => setSemester(event.target.value)}
            hint={running ? `${semesterLabel(running)} is running now` : undefined}
          >
            {semesterOptions(running).map((entry) => (
              <option key={entry} value={entry}>
                {semesterLabel(entry)}
                {entry === running ? ' · running now' : ''}
              </option>
            ))}
          </Select>
        </div>
      </Card>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : (
        <>
          <StatusTabs tabs={SECTION_TABS} value={tab} onChange={setTab} className="mb-4 lg:hidden" />
          <div className="lg:hidden">
            <RoutineWeekGrid items={items} periods={periods} section={tab} />
          </div>
          <div className="hidden gap-4 lg:grid lg:grid-cols-2">
            {SECTIONS.map((section) => (
              <RoutineWeekGrid key={section} items={items} periods={periods} section={section} />
            ))}
          </div>
        </>
      )}
    </>
  )
}
