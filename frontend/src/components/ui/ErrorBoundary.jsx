import { Component } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { CircleAlert } from 'lucide-react'
import api from '../../lib/api'
import { CRASH_PATH, crashReport, createCrashReporter } from '../../lib/crash'
import cx from '../../lib/cx'
import { isNative } from '../../lib/native'
import paths from '../../lib/paths'
import { APP_VERSION } from '../../lib/version'
import Button from './Button'

const report = createCrashReporter((body) => api.post(CRASH_PATH, body))

const tell = (error) => {
  const client = isNative() ? 'app' : 'web'
  report(crashReport(error, window.location, client, APP_VERSION))
}

export function CrashScreen({ onRetry, onHome, onClose, compact = false }) {
  return (
    <div
      role="alert"
      data-crash-screen
      className={cx(
        'mx-auto flex w-full max-w-md flex-col items-center gap-3 px-5 text-center',
        compact ? 'py-6' : 'min-h-[60dvh] justify-center py-10',
      )}
    >
      <span className="grid size-14 place-items-center rounded-full bg-wash text-brand-600">
        <CircleAlert size={26} aria-hidden="true" />
      </span>
      <h2 className="font-display text-section font-bold text-ink">Something went wrong</h2>
      <p className="text-sm leading-relaxed text-muted">
        This screen could not be shown. Nothing you saved is lost, and the problem has been
        reported.
      </p>
      <div className="mt-2 flex flex-wrap justify-center gap-2">
        <Button onClick={onRetry}>Try again</Button>
        {onHome && (
          <Button variant="secondary" onClick={onHome}>
            Go home
          </Button>
        )}
        {onClose && (
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
        )}
      </div>
    </div>
  )
}

class Boundary extends Component {
  constructor(props) {
    super(props)
    this.state = { error: null }
    this.retry = this.retry.bind(this)
    this.home = this.home.bind(this)
  }

  static getDerivedStateFromError(error) {
    return { error }
  }

  componentDidCatch(error) {
    tell(error)
  }

  componentDidUpdate(previous) {
    if (this.state.error && previous.resetKey !== this.props.resetKey) this.setState({ error: null })
  }

  retry() {
    this.setState({ error: null })
  }

  home() {
    this.setState({ error: null })
    this.props.onHome?.()
  }

  render() {
    if (!this.state.error) return this.props.children
    return (
      <CrashScreen
        compact={this.props.compact}
        onRetry={this.retry}
        onHome={this.props.onHome ? this.home : null}
        onClose={this.props.onClose}
      />
    )
  }
}

export function SheetBoundary({ onClose, children }) {
  return (
    <Boundary compact onClose={onClose}>
      {children}
    </Boundary>
  )
}

export default function ErrorBoundary({ compact = false, children }) {
  const location = useLocation()
  const navigate = useNavigate()
  return (
    <Boundary
      compact={compact}
      resetKey={`${location.pathname}${location.search}`}
      onHome={() => navigate(paths.root())}
    >
      {children}
    </Boundary>
  )
}
