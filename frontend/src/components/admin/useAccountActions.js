import { useState } from 'react'
import api from '../../lib/api'
import { useConfirm } from '../ui/confirm'
import { useToast } from '../ui/toast'

export const useAccountActions = (reload) => {
  const toast = useToast()
  const confirm = useConfirm()
  const [busy, setBusy] = useState(0)
  const [deleting, setDeleting] = useState(null)
  const restore = async (account) => {
    const ok = await confirm({
      title: 'Restore this account?',
      message:
        account.status === 'rejected'
          ? `${account.name} goes back to the pending registrations.`
          : `${account.name} can sign in again and has to log in from scratch.`,
      confirmLabel: 'Restore',
    })
    if (!ok) return
    setBusy(account.id)
    try {
      await api.post(`/api/admin/users/${account.id}/restore`)
      toast.success('Account restored')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const deletePermanently = async (confirmText) => {
    setBusy(deleting.id)
    try {
      await api.delete(`/api/admin/users/${deleting.id}/permanent`, { body: { confirm: confirmText } })
      toast.success('Account deleted permanently')
      setDeleting(null)
      reload()
    } finally {
      setBusy(0)
    }
  }
  return { busy, deleting, setDeleting, restore, deletePermanently }
}

export default useAccountActions
