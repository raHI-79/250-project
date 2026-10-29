import { Smartphone } from 'lucide-react'
import Button from '../ui/Button'
import { LATEST_APK, showsAndroidDownload } from '../../lib/native'

export default function AndroidDownload({ variant = 'secondary', size = 'lg', className }) {
  if (!showsAndroidDownload()) return null
  return (
    <Button
      as="a"
      href={LATEST_APK}
      target="_blank"
      rel="noreferrer"
      variant={variant}
      size={size}
      className={className}
    >
      <Smartphone size={18} />
      Download the Android app
    </Button>
  )
}
