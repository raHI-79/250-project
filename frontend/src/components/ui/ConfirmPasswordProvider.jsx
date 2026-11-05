import { useCallback, useEffect, useRef, useState } from 'react'
import { KeyRound, Lock } from 'lucide-react'
import Alert from './Alert'
import Button from './Button'
import Modal from './Modal'
import { Input, PasswordInput } from './Field'
import { Cancelled, setConfirmAsker } from '../../lib/confirmgate'

const CONFIRM_PATH = '/api/office/security/confirm'

const EMPTY = { password: '', code: '' }

export default function ConfirmPasswordProvider({ children }) {
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(EMPTY)
  const [needsCode, setNeedsCode] = useState(false)
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const settle = useRef(null)
  const finish = useCallback((result, failure) => {
    const waiting = settle.current
    settle.current = null
    setOpen(false)
    setForm(EMPTY)
    setNeedsCode(false)
    setNote('')
    setError('')
    setBusy(false)
    if (!waiting) return
    if (failure) waiting.reject(failure)
    else waiting.resolve(result)
  }, [])
  useEffect(() => {
    setConfirmAsker(
      () =>
        new Promise((resolve, reject) => {
          settle.current = { resolve, reject }
          setForm(EMPTY)
          setNeedsCode(false)
          setNote('')
          setError('')
          setOpen(true)
        }),
    )
    return () => setConfirmAsker(null)
  }, [])
  const confirm = async (event) => {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      const { default: api } = await import('../../lib/api')
      const data = await api.post(CONFIRM_PATH, {
        password: form.password,
        code: form.code.trim(),
      })
      if (data?.needs_code) {
        setNeedsCode(true)
        setNote(data.message || 'We sent a code to your email address.')
        setBusy(false)
        return
      }
      finish(data)
    } catch (failure) {
      setError(failure.message)
      setBusy(false)
    }
  }
  return (
    <>
      {children}
      <Modal
        open={open}
        onClose={() => finish(null, new Cancelled())}
        title="Confirm it's you"
        footer={
          <>
            <Button variant="secondary" onClick={() => finish(null, new Cancelled())}>
              Cancel
            </Button>
            <Button onClick={confirm} loading={busy} disabled={!form.password}>
              Confirm
            </Button>
          </>
        }
      >
        <form onSubmit={confirm} className="flex flex-col gap-3">
          <p className="text-sm text-ink-soft">
            This action changes something sensitive. Type your password to go on; the confirmation
            lasts five minutes.
          </p>
          {note && <Alert tone="info">{note}</Alert>}
          {error && <Alert tone="error">{error}</Alert>}
          <PasswordInput
            label="Your password"
            icon={Lock}
            value={form.password}
            onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))}
            autoComplete="current-password"
            required
          />
          {needsCode && (
            <Input
              label="6-digit code"
              icon={KeyRound}
              value={form.code}
              onChange={(event) => setForm((current) => ({ ...current, code: event.target.value }))}
              inputMode="numeric"
              autoComplete="one-time-code"
            />
          )}
        </form>
      </Modal>
    </>
  )
}
