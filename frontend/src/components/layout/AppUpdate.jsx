import { useCallback, useEffect, useRef, useState } from 'react'
import { Download, ShieldCheck } from 'lucide-react'
import { Alert, Button, Sheet, useToast } from '../ui'
import {
  canInstall,
  downloadUpdate,
  fetchLatest,
  guidedOnce,
  isNewer,
  markGuided,
  openInstallSettings,
  openInstaller,
  updatesSupported,
  CHECK_EVENT,
} from '../../lib/update'
import { APP_VERSION } from '../../lib/version'

const BUSY_NOTE = 'Could not reach the update service. Please try again.'

const ALLOW_NOTE =
  'Android asks once for permission. Turn on "Allow from this source" on the screen that opens, come back and tap Update again.'

export default function AppUpdate() {
  const toast = useToast()
  const [latest, setLatest] = useState(null)
  const [stage, setStage] = useState('idle')
  const [percent, setPercent] = useState(0)
  const [failure, setFailure] = useState('')
  const [guide, setGuide] = useState(false)
  const busy = useRef(false)
  const check = useCallback(
    async (announce) => {
      if (!updatesSupported() || busy.current) return
      busy.current = true
      try {
        const details = await fetchLatest()
        if (details?.apk_url && isNewer(details.version)) {
          setLatest(details)
          setFailure('')
          return
        }
        if (announce) toast.success(`You are on the latest version, ${APP_VERSION}`)
      } catch (reason) {
        if (announce) toast.info(reason?.status === 503 ? reason.message : BUSY_NOTE)
      } finally {
        busy.current = false
      }
    },
    [toast],
  )
  useEffect(() => {
    if (!updatesSupported()) return undefined
    let live = true
    let listener = null
    check(false)
    const onRequest = () => {
      if (live) check(true)
    }
    window.addEventListener(CHECK_EVENT, onRequest)
    import('@capacitor/app').then(({ App }) => {
      if (!live) return
      App.addListener('appStateChange', ({ isActive }) => {
        if (isActive && live) check(false)
      }).then((handle) => {
        if (live) listener = handle
        else handle.remove()
      })
    })
    return () => {
      live = false
      window.removeEventListener(CHECK_EVENT, onRequest)
      if (listener) listener.remove()
    }
  }, [check])
  if (!latest) return null
  const required = Boolean(latest.required)
  const running = stage === 'downloading'
  const close = () => {
    setLatest(null)
    setStage('idle')
    setPercent(0)
    setFailure('')
    setGuide(false)
  }
  const update = async () => {
    setFailure('')
    setGuide(false)
    try {
      if (!(await canInstall())) {
        setGuide(true)
        if (!guidedOnce()) markGuided()
        await openInstallSettings()
        return
      }
      setStage('downloading')
      setPercent(0)
      const path = await downloadUpdate(latest.apk_url, latest.version, setPercent)
      if (!path) throw new Error('The download did not finish')
      setStage('installing')
      await openInstaller(path)
    } catch (error) {
      setStage('idle')
      setFailure(error?.message || 'The update could not be installed. Please try again.')
    }
  }
  return (
    <Sheet
      open
      onClose={required ? undefined : close}
      title={`Update available · ${latest.version}`}
      description={`You are on ${APP_VERSION}`}
      footer={
        <>
          {!required && (
            <Button variant="secondary" onClick={close} disabled={running}>
              Later
            </Button>
          )}
          <Button onClick={update} loading={running}>
            <Download size={16} />
            {running ? `Downloading ${percent}%` : 'Update'}
          </Button>
        </>
      }
    >
      {required && (
        <Alert tone="warning" className="mb-4">
          This update is required to keep using the app.
        </Alert>
      )}
      {failure && (
        <Alert tone="error" className="mb-4">
          {failure}
        </Alert>
      )}
      {guide && (
        <Alert tone="info" className="mb-4">
          {ALLOW_NOTE}
        </Alert>
      )}
      {latest.notes ? (
        <p className="whitespace-pre-line text-sm leading-relaxed text-ink-soft">{latest.notes}</p>
      ) : (
        <p className="text-sm text-muted">Fixes and small improvements.</p>
      )}
      {running && (
        <div className="mt-5">
          <div className="h-1.5 overflow-hidden rounded-full bg-wash">
            <div
              className="h-full rounded-full bg-brand-600 transition-[width] duration-300"
              style={{ width: `${percent}%` }}
            />
          </div>
          <p className="mt-2 text-xs text-muted">Downloading the update · {percent}%</p>
        </div>
      )}
      {stage === 'installing' && (
        <p className="mt-5 flex items-center gap-2 text-xs text-muted">
          <ShieldCheck size={15} className="shrink-0 text-success-600" />
          Android is installing the update over this app. Your sign-in stays.
        </p>
      )}
    </Sheet>
  )
}
