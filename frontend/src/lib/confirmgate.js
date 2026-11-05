export const CONFIRM_STATUS = 428

export const CONFIRM_MESSAGE = 'Please confirm your password first.'

export const CONFIRM_WINDOW = 5 * 60 * 1000

export const CANCELLED = 'confirm-cancelled'

export class Cancelled extends Error {
  constructor() {
    super('')
    this.name = 'ConfirmCancelled'
    this.cancelled = true
  }
}

export const isCancelled = (failure) => Boolean(failure?.cancelled)

export const needsConfirm = (failure) =>
  Boolean(failure) &&
  failure.status === CONFIRM_STATUS &&
  String(failure.message || '').startsWith('Please confirm')

let ask = null

let waiting = null

let until = 0

const listeners = new Set()

const announce = () => {
  listeners.forEach((listener) => listener(until))
}

export const setConfirmAsker = (handler) => {
  ask = handler
}

export const confirmedUntil = () => (until > Date.now() ? until : 0)

export const markConfirmed = (at) => {
  until = Number(at) || Date.now() + CONFIRM_WINDOW
  announce()
}

export const forgetConfirmed = () => {
  until = 0
  announce()
}

export const watchConfirmed = (listener) => {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export const askOnce = () => {
  if (waiting) return waiting
  if (!ask) return Promise.reject(new Cancelled())
  let started
  try {
    started = Promise.resolve(ask())
  } catch (failure) {
    return Promise.reject(failure)
  }
  waiting = started
    .then((result) => {
      markConfirmed(result?.until ? new Date(result.until).getTime() : 0)
      return result
    })
    .finally(() => {
      waiting = null
    })
  return waiting
}

export const withConfirm = async (run, confirm = askOnce) => {
  try {
    return await run()
  } catch (failure) {
    if (!needsConfirm(failure)) throw failure
    await confirm()
    return run()
  }
}
