import PhoneRequestList from '../../components/attendance/PhoneRequestList'
import { PageHeader, PullToRefresh } from '../../components/ui'

export default function TeacherPhoneRequestsPage() {
  return (
    <PullToRefresh onRefresh={() => Promise.resolve()}>
      <PageHeader
        eyebrow="Your students"
        title="Phone requests"
        subtitle="Students of the batches you teach asking to move their attendance to another phone. Any one of their teachers can decide."
      />
      <PhoneRequestList />
    </PullToRefresh>
  )
}
