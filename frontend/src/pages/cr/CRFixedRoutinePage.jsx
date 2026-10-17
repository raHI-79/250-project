import { BookOpen } from 'lucide-react'
import { Alert, EmptyState, PageHeader, SkeletonCard } from '../../components/ui'
import RoutineByDay from '../../components/RoutineByDay'
import { useAuth } from '../../context/auth'
import { useResource } from '../../lib/hooks'

export default function CRFixedRoutinePage() {
  const { user } = useAuth()
  const { data, loading, error } = useResource('/api/cr/routine')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Fixed Routine"
        subtitle={
          user?.batch
            ? `The weekly routine of ${user.batch} section ${user.section}`
            : 'The weekly routine of your batch and section'
        }
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={BookOpen}
          title="No routine entries"
          text="The administration has not published your routine yet."
        />
      ) : (
        <RoutineByDay items={items} />
      )}
    </>
  )
}
