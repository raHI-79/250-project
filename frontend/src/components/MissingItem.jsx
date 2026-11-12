import { Link } from 'react-router-dom'
import { Alert } from './ui'

export default function MissingItem({ highlight, what, all, to }) {
  if (!highlight?.missing) return null
  return (
    <Alert tone="info" className="mb-5" data-missing-item>
      This {what} is no longer available.{' '}
      <Link to={to} replace className="font-semibold underline">
        {all}
      </Link>
    </Alert>
  )
}
