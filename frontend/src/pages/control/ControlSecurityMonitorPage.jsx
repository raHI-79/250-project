import { useState } from 'react'
import { Ban, Lock, ShieldAlert } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  SkeletonLines,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { activityLabel } from '../../lib/options'
import { formatDateTime } from '../../lib/time'

export default function ControlSecurityMonitorPage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/super-admin/security')
  const [cidr, setCidr] = useState('')
  const [hours, setHours] = useState('24')
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState('')
  const act = async (key, call) => {
    setBusy(key)
    try {
      const result = await call()
      toast(result.message)
      await reload()
    } catch (failure) {
      toast(failure.message, 'error')
    } finally {
      setBusy('')
    }
  }
  const block = (event) => {
    event.preventDefault()
    act('block', () =>
      api.post('/api/super-admin/security/blocks', {
        cidr: cidr.trim(),
        reason: reason.trim(),
        hours: Number.parseInt(hours, 10) || 0,
      }),
    ).then(() => {
      setCidr('')
      setReason('')
    })
  }
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Security"
        subtitle="Failed sign-ins, locked accounts and blocked addresses over the last week"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonLines lines={6} />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card hover={false} flush>
            <CardHeader title="Failed sign-ins by address" subtitle="The hash of the address, never the address itself." />
            {(data?.addresses || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={ShieldAlert} title="Nothing to see" text="No failed sign-ins this week." />
              </div>
            ) : (
              <DataTable>
                <tbody>
                  {data.addresses.map((row) => (
                    <tr key={row.label} className="border-b border-line last:border-b-0">
                      <td className="px-4 py-2 font-mono text-xs text-ink-soft">{row.label}…</td>
                      <td className="px-4 py-2 text-right font-semibold text-ink">{row.total}</td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
          <Card hover={false} flush>
            <CardHeader title="Failed sign-ins by account" subtitle="The identifier that was tried." />
            {(data?.accounts || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={ShieldAlert} title="Nothing to see" text="No failed sign-ins this week." />
              </div>
            ) : (
              <DataTable>
                <tbody>
                  {data.accounts.map((row) => (
                    <tr key={row.label} className="border-b border-line last:border-b-0">
                      <td className="px-4 py-2 text-ink-soft">{row.label}</td>
                      <td className="px-4 py-2 text-right font-semibold text-ink">{row.total}</td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
          <Card hover={false} flush>
            <CardHeader title="Locked accounts" subtitle="Locked after five wrong passwords." />
            {(data?.locked || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={Lock} title="No locked accounts" text="Everybody can sign in." />
              </div>
            ) : (
              <DataTable>
                <tbody>
                  {data.locked.map((row) => (
                    <tr key={row.id} className="border-b border-line last:border-b-0">
                      <td className="px-4 py-2">
                        <p className="font-semibold text-ink">{row.identifier}</p>
                        <p className="text-xs text-muted">
                          {activityLabel(row.role)} · {row.failures} tries · until{' '}
                          {formatDateTime(row.locked_until)}
                        </p>
                      </td>
                      <td className="px-4 py-2 text-right">
                        <Button
                          size="sm"
                          variant="secondary"
                          loading={busy === `unlock-${row.id}`}
                          onClick={() =>
                            act(`unlock-${row.id}`, () =>
                              api.post(`/api/super-admin/security/unlock/${row.id}`),
                            )
                          }
                        >
                          Unlock
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
          <Card hover={false}>
            <CardHeader title="Blocked addresses" subtitle="A single address or a range like 203.0.113.0/24." />
            <form onSubmit={block} className="mt-4 flex flex-col gap-3">
              <Input
                label="Address or range"
                value={cidr}
                onChange={(event) => setCidr(event.target.value)}
                required
              />
              <div className="grid gap-3 sm:grid-cols-2">
                <Input
                  label="Hours"
                  value={hours}
                  onChange={(event) => setHours(event.target.value)}
                  inputMode="numeric"
                  hint="0 keeps it forever"
                />
                <Input
                  label="Reason"
                  value={reason}
                  onChange={(event) => setReason(event.target.value)}
                />
              </div>
              <Button type="submit" variant="danger" loading={busy === 'block'}>
                <Ban size={16} aria-hidden="true" />
                Block
              </Button>
            </form>
            {(data?.blocks || []).length > 0 && (
              <ul className="mt-4 flex flex-col divide-y divide-line">
                {data.blocks.map((row) => (
                  <li key={row.id} className="flex items-center justify-between gap-3 py-2">
                    <div className="min-w-0">
                      <p className="font-mono text-sm text-ink">{row.cidr}</p>
                      <p className="text-xs text-muted">
                        {row.reason || 'No reason given'}
                        {row.expires_at ? ` · until ${formatDateTime(row.expires_at)}` : ' · forever'}
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant="secondary"
                      loading={busy === `unblock-${row.id}`}
                      onClick={() =>
                        act(`unblock-${row.id}`, () =>
                          api.delete(`/api/super-admin/security/blocks/${row.id}`),
                        )
                      }
                    >
                      Unblock
                    </Button>
                  </li>
                ))}
              </ul>
            )}
          </Card>
          <Card hover={false} flush className="lg:col-span-2">
            <CardHeader title="Latest failed sign-ins" subtitle="Newest first." />
            {(data?.events || []).length === 0 ? (
              <div className="px-4 pb-4">
                <EmptyState icon={ShieldAlert} title="Nothing recorded" text="No failed sign-in yet." />
              </div>
            ) : (
              <DataTable>
                <tbody>
                  {data.events.map((row) => (
                    <tr key={row.id} className="border-b border-line last:border-b-0">
                      <td className="px-4 py-2 whitespace-nowrap text-xs text-muted">
                        {formatDateTime(row.created_at)}
                      </td>
                      <td className="px-4 py-2 wrap-long text-sm text-ink">{row.summary}</td>
                      <td className="px-4 py-2 text-xs text-muted">
                        {row.client === 'android' ? 'App' : 'Website'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
        </div>
      )}
    </>
  )
}
