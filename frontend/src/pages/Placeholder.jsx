import { Construction } from 'lucide-react'
import EmptyState from '../components/ui/EmptyState'
import PageHeader from '../components/ui/PageHeader'

export default function Placeholder({ title, subtitle, text }) {
  return (
    <>
      <PageHeader title={title} subtitle={subtitle} />
      <EmptyState
        icon={Construction}
        title="Coming soon"
        text={text || 'This page is part of the next development step.'}
      />
    </>
  )
}
