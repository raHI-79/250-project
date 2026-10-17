import { createContext, useContext } from 'react'

export const ConfirmContext = createContext(null)

export const useConfirm = () => {
  const value = useContext(ConfirmContext)
  if (!value) throw new Error('useConfirm must be used inside ConfirmProvider')
  return value
}
