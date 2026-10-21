import { useEffect, useState } from 'react'
import { Alert, Button, Input, Modal } from '../ui'

const KEPT = 'Notices, messages, class requests, polls and routines stay, with the author shown as "Deleted user".'

const REMOVED =
  'The account, the profile, the dropped-course rows, the poll votes, the conversation memberships and the profile photo are deleted.'

export default function DeleteAccountModal({ account, onClose, onConfirmed, busy }) {
  const [typed, setTyped] = useState('')
  const [problem, setProblem] = useState('')
  useEffect(() => {
    setTyped('')
    setProblem('')
  }, [account])
  const expected = account?.identifier || account?.email || ''
  const matches = Boolean(expected) && typed.trim().toLowerCase() === expected.toLowerCase()
  const confirm = async () => {
    setProblem('')
    try {
      await onConfirmed(typed.trim())
    } catch (failure) {
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(account)}
      onClose={onClose}
      title="Delete this account permanently?"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" disabled={!matches} loading={busy} onClick={confirm}>
            Delete permanently
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        <Alert tone="error" title="This cannot be undone">
          {REMOVED}
        </Alert>
        <Alert tone="info" title="What stays">
          {KEPT}
        </Alert>
        <Input
          label={`Type ${expected} to confirm`}
          value={typed}
          onChange={(event) => setTyped(event.target.value)}
          placeholder={expected}
          autoComplete="off"
        />
      </div>
    </Modal>
  )
}
