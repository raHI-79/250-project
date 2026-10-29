import { Smartphone } from 'lucide-react'
import Button from '../ui/Button'
import paths from '../../lib/paths'
import { showsAndroidDownload } from '../../lib/native'

export default function AndroidDownload({ variant = 'secondary', size = 'lg', className }) {
  if (!showsAndroidDownload()) return null
  return (
    <Button
      as="a"
      href={paths.downloadAndroid()}
      variant={variant}
      size={size}
      className={className}
    >
      <Smartphone size={18} />
      Download the Android app
    </Button>
  )
}
