import { useCallback, useRef, useState } from 'react'
import Button from './Button'
import Modal from './Modal'
import { ConfirmContext } from './confirm'

const EMPTY = {
  title: 'Are you sure?',
  message: '',
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  tone: 'danger',
}

export default function ConfirmProvider({ children }) {
  const [state, setState] = useState(null)
  const resolver = useRef(null)
  const settle = useCallback((result) => {
    setState(null)
    const resolve = resolver.current
    resolver.current = null
    resolve?.(result)
  }, [])
  const confirm = useCallback((options = {}) => {
    const next = typeof options === 'string' ? { message: options } : options
    setState({ ...EMPTY, ...next })
    return new Promise((resolve) => {
      resolver.current = resolve
    })
  }, [])
  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      <Modal
        open={Boolean(state)}
        onClose={() => settle(false)}
        title={state?.title}
        footer={
          <>
            {state?.cancelLabel && (
              <Button variant="secondary" onClick={() => settle(false)}>
                {state.cancelLabel}
              </Button>
            )}
            <Button variant={state?.tone === 'danger' ? 'danger' : 'primary'} onClick={() => settle(true)}>
              {state?.confirmLabel}
            </Button>
          </>
        }
      >
        {state?.message}
      </Modal>
    </ConfirmContext.Provider>
  )
}
