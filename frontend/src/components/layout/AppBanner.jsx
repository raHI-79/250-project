import { useState } from 'react'
import { Download, X } from 'lucide-react'
import Button from '../ui/Button'
import { SUST_MARK } from '../../lib/brand'
import { isAndroidBrowser } from '../../lib/native'
import paths from '../../lib/paths'
import { readRaw, writeRaw } from '../../lib/storage'

const KEY = 'sustcse.appbanner'

const HIDE_FOR = 30 * 24 * 60 * 60 * 1000

const dismissed = () => {
  const until = Number(readRaw(KEY))
  return Number.isFinite(until) && until > Date.now()
}

export default function AppBanner() {
  const [shown, setShown] = useState(() => isAndroidBrowser() && !dismissed())
  const close = () => {
    writeRaw(KEY, String(Date.now() + HIDE_FOR))
    setShown(false)
  }
  if (!shown) return null
  return (
    <div
      role="complementary"
      aria-label="Get the SUST CSE app"
      className="glass-sheet animate-fade-in-up fixed inset-x-3 bottom-[calc(4.75rem+env(safe-area-inset-bottom,0px))] z-70 flex items-center gap-3 rounded-card px-3 py-2.5 lg:hidden"
    >
      <img src={SUST_MARK} alt="" className="size-9 shrink-0 object-contain" />
      <p className="min-w-0 flex-1 text-sm font-semibold text-ink">Get the SUST CSE app</p>
      <Button as="a" href={paths.downloadAndroid()} size="sm" onClick={close}>
        <Download size={15} />
        Download
      </Button>
      <button
        type="button"
        onClick={close}
        aria-label="Hide this"
        className="press -mr-1 grid size-9 shrink-0 place-items-center rounded-full text-muted hover:bg-wash hover:text-ink"
      >
        <X size={17} />
      </button>
    </div>
  )
}
