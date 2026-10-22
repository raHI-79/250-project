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
    <div className="animate-fade-in mb-6 flex items-start gap-3 border border-gold-200 border-l-[3px] border-l-gold-500 bg-gold-50 px-4 py-3">
      <span className="mt-0.5 shrink-0 text-gold-600">
        <ImagePlus size={18} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm text-gold-800">
          Add a profile photo so others can recognise you
        </p>
        <Link
          to={paths.profile()}
          className="mt-0.5 inline-block text-sm font-semibold text-gold-800 underline"
        >
          Go to My Profile
        </Link>
      </div>
      <button
        type="button"
        onClick={hide}
        aria-label="Dismiss this reminder"
        className="shrink-0 p-1 text-gold-700 transition-colors hover:bg-gold-100"
      >
        <X size={16} />
      </button>
    </div>
  )
}
