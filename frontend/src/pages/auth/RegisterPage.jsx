import { Link } from 'react-router-dom'
import Card from '../../components/ui/Card'
import EmptyState from '../../components/ui/EmptyState'
import { paths } from '../../lib/paths'

export default function RegisterPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-page px-4 py-10">
      <Card hover={false} className="animate-fade-in-up w-full max-w-md">
        <EmptyState
          title="Registration is on the way"
          text="The student registration form arrives with the next development step."
          className="border-0"
          action={
            <Link to={paths.login()} className="text-sm text-brand-600 hover:underline">
              Back to log in
            </Link>
          }
        />
      </Card>
    </div>
  )
}
