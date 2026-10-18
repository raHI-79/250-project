import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ImagePlus, X } from 'lucide-react'
import { useAuth } from '../../context/auth'
import { readRaw, writeRaw } from '../../lib/storage'
import { paths } from '../../lib/paths'

const keyFor = (userId) => `sustcse.photoReminder.${userId}`

export default function PhotoReminder() {
  const { user, isAuthenticated } = useAuth()
  const userId = user?.user_id || user?.id || 0
  const [dismissed, setDismissed] = useState(() =>
    userId ? readRaw(keyFor(userId)) === 'hidden' : true,
  )
  if (!isAuthenticated || !userId || user?.photo_url || dismissed) return null
  const hide = () => {
    writeRaw(keyFor(userId), 'hidden')
    setDismissed(true)
  }
  return (
    <div className="animate-fade-in mb-6 flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3">
      <span className="mt-0.5 shrink-0 text-amber-600">
        <ImagePlus size={18} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm text-amber-900">
          Add a profile photo so others can recognise you
        </p>
        <Link
          to={paths.profile()}
          className="mt-0.5 inline-block text-sm font-medium text-amber-900 underline"
        >
          Go to My Profile
        </Link>
      </div>
      <button
        type="button"
        onClick={hide}
        aria-label="Dismiss this reminder"
        className="shrink-0 rounded p-1 text-amber-700 transition-colors hover:bg-amber-100"
      >
        <X size={16} />
      </button>
    </div>
  )
}
