import { useEffect } from 'react'
import { useAuth } from '../../context/auth'
import { askPushPermission, closePushAsk, retryAskPending, usePushSync } from '../../lib/push'

export default function PushGate() {
  const { role } = useAuth()
  usePushSync(role)
  useEffect(() => {
    if (!retryAskPending(role)) return undefined
    const onTap = () => {
      document.removeEventListener('pointerdown', onTap)
      closePushAsk()
      askPushPermission(role).catch(() => null)
    }
    document.addEventListener('pointerdown', onTap, { once: true })
    return () => document.removeEventListener('pointerdown', onTap)
  }, [role])
  return null
}
