import { Link } from 'react-router-dom'
import { Compass } from 'lucide-react'
import Card from '../components/ui/Card'
import EmptyState from '../components/ui/EmptyState'
import { useAuth } from '../context/auth'
import { homeFor } from '../lib/paths'

export default function NotFoundPage() {
  const { role } = useAuth()
  return (
    <div className="flex min-h-screen items-center justify-center bg-page px-4 py-10">
      <Card hover={false} className="w-full max-w-md">
        <EmptyState
          icon={Compass}
          title="Page not found"
          text="The page you asked for does not exist."
          className="border-0"
          action={
            <Link to={homeFor(role)} className="text-sm text-brand-600 hover:underline">
              Go back
            </Link>
          }
        />
      </Card>
    </div>
  )
}
