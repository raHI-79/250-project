import { useState } from 'react'
import { KeyRound } from 'lucide-react'
import Alert from '../ui/Alert'
import Button from '../ui/Button'
import { Input } from '../ui/Field'

export default function TwoStepForm({ challenge, onSubmit, onCancel }) {
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const send = async (event) => {
    event.preventDefault()
    setError('')
    setLoading(true)
    try {
      await onSubmit(code.trim())
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  return (
    <form onSubmit={send} className="mt-6 flex flex-col gap-4">
      <Alert tone="info" title="One more step">
        {challenge.message}
        {challenge.email_hint ? ` (${challenge.email_hint})` : ''}
      </Alert>
      {challenge.dev_code && (
        <Alert tone="info" title="Development code">
          {challenge.dev_code}
        </Alert>
      )}
      {error && <Alert tone="error">{error}</Alert>}
      <Input
        label="6-digit code"
        icon={KeyRound}
        value={code}
        onChange={(event) => setCode(event.target.value)}
        inputMode="numeric"
        autoComplete="one-time-code"
        required
      />
      <Button type="submit" loading={loading} className="w-full">
        Sign in
      </Button>
      <Button type="button" variant="secondary" onClick={onCancel} className="w-full">
        Start again
      </Button>
    </form>
  )
}
