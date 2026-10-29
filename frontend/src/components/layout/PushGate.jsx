import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/auth'
import { useFCM } from '../../lib/fcm'
import { isAndroidApp } from '../../lib/native'
import { askPushPermission, closePushAsk, retryAskPending, usePushSync } from '../../lib/push'

export default function PushGate() {
  const { role } = useAuth()
  const navigate = useNavigate()
  const native = isAndroidApp()
  usePushSync(native ? null : role)
  useFCM(role, navigate)
  useEffect(() => {
    if (native || !retryAskPending(role)) return undefined
    const onTap = () => {
      document.removeEventListener('pointerdown', onTap)
      closePushAsk()
      askPushPermission(role).catch(() => null)
    }
    document.addEventListener('pointerdown', onTap, { once: true })
    return () => document.removeEventListener('pointerdown', onTap)
  }, [native, role])
  return null
}
