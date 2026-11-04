import MyAttendance from '../../components/attendance/MyAttendance'
import { PageHeader, PullToRefresh } from '../../components/ui'

export default function MyAttendancePage() {
  return (
    <PullToRefresh onRefresh={() => Promise.resolve()}>
      <PageHeader
        eyebrow="Your classes"
        title="My attendance"
        subtitle="Your share of every course, and every class with the time, the teacher and any note."
      />
      <MyAttendance />
    </PullToRefresh>
  )
}
