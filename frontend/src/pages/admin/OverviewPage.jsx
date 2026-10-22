import { GraduationCap, UserCheck, UserCog, Users } from 'lucide-react'
import { Alert, PageHeader, SkeletonCard } from '../../components/ui'
import StatCard from '../../components/StatCard'
import Masthead from '../../components/layout/Masthead'
import { useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'

export default function OverviewPage() {
  const { data, loading, error } = useResource('/api/admin/stats')
  const cards = [
    { label: 'Total students', value: data?.total_students, icon: Users },
    { label: 'CRs', value: data?.total_crs, icon: UserCog },
    { label: 'Teachers', value: data?.total_teachers, icon: GraduationCap },
    {
      label: 'Pending approvals',
      value: data?.pending_approvals,
      icon: UserCheck,
      to: paths.admin.registrations(),
      hint: 'Open the registrations',
    },
  ]
  return (
    <>
      <Masthead subtitle="How the department is doing right now" />
      <PageHeader eyebrow="Department office" title="Overview" />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {cards.map((card) => (
            <SkeletonCard key={card.label} />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {cards.map((card) => (
            <StatCard
              key={card.label}
              icon={card.icon}
              label={card.label}
              value={card.value || 0}
              to={card.to}
              hint={card.hint}
            />
          ))}
        </div>
      )}
    </>
  )
}
