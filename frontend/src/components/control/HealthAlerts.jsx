import { useState } from 'react'
import { Alert, Button, useToast } from '../ui'
import api from '../../lib/api'

const TONES = { error: 'error', warning: 'warning', info: 'info' }

export default function HealthAlerts({ alerts = [], onDismissed, className }) {
  const toast = useToast()
  const [busy, setBusy] = useState('')
  if (alerts.length === 0) return null
  const dismiss = async (alert) => {
    setBusy(alert.key)
    try {
      await api.post(`/api/super-admin/alerts/${alert.key}/dismiss`)
      toast.success('The alert stays quiet for a day')
      onDismissed?.()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  return (
    <div className={className}>
      <div className="flex flex-col gap-2">
        {alerts.map((alert) => (
          <Alert key={alert.key} tone={TONES[alert.tone] || 'warning'} data-alert={alert.key}>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
              <span className="min-w-0 wrap-long">
                {alert.message}
                {(alert.details || []).length > 0 && (
                  <span className="mt-1 block text-xs">
                    {alert.details.map((entry) => `${entry.label} (${entry.count})`).join(' · ')}
                  </span>
                )}
              </span>
              <span className="ml-auto flex shrink-0 items-center gap-2">
                {alert.section && (
                  <a
                    href={`#system-${alert.section}`}
                    className="text-sm font-semibold underline hover:no-underline"
                  >
                    Details
                  </a>
                )}
                <Button
                  size="sm"
                  variant="secondary"
                  loading={busy === alert.key}
                  onClick={() => dismiss(alert)}
                >
                  Dismiss for a day
                </Button>
              </span>
            </div>
          </Alert>
        ))}
      </div>
    </div>
  )
}
