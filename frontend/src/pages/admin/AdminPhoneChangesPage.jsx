import AdminPhoneChanges from '../../components/attendance/AdminPhoneChanges'
import { PageHeader } from '../../components/ui'

export default function AdminPhoneChangesPage() {
  return (
    <>
      <PageHeader
        eyebrow="Department office"
        title="Phone changes"
        subtitle="Every attendance phone a student asked for and what their teachers decided. The office does not approve these."
      />
      <AdminPhoneChanges />
    </>
  )
}
