import assert from 'node:assert/strict'
import test from 'node:test'
import {
  CONFIRM_STATUS,
  Cancelled,
  confirmedUntil,
  forgetConfirmed,
  isCancelled,
  markConfirmed,
  needsConfirm,
  setConfirmAsker,
  askOnce,
  withConfirm,
} from './confirmgate.js'

const confirmFailure = () => {
  const failure = new Error('Please confirm your password first.')
  failure.status = CONFIRM_STATUS
  return failure
}

test('only the confirmation rule starts the dialog', () => {
  assert.equal(needsConfirm(confirmFailure()), true)
  const other = new Error('You do not have permission for this')
  other.status = 403
  assert.equal(needsConfirm(other), false)
  const same = new Error('Please confirm your password first.')
  same.status = 400
  assert.equal(needsConfirm(same), false)
  assert.equal(needsConfirm(null), false)
})

test('a 428 is retried once after the confirmation', async () => {
  let calls = 0
  let confirmed = 0
  const run = async () => {
    calls += 1
    if (calls === 1) throw confirmFailure()
    return 'done'
  }
  const answer = await withConfirm(run, async () => {
    confirmed += 1
  })
  assert.equal(answer, 'done')
  assert.equal(calls, 2)
  assert.equal(confirmed, 1)
})

test('a wrong password keeps the error and does not retry', async () => {
  let calls = 0
  const run = async () => {
    calls += 1
    throw confirmFailure()
  }
  await assert.rejects(
    () => withConfirm(run, async () => {
      throw new Error('That password is wrong.')
    }),
    /That password is wrong\./,
  )
  assert.equal(calls, 1)
})

test('cancelling rejects quietly with no message', async () => {
  const run = async () => {
    throw confirmFailure()
  }
  await assert.rejects(
    () => withConfirm(run, async () => {
      throw new Cancelled()
    }),
    (failure) => {
      assert.equal(isCancelled(failure), true)
      assert.equal(failure.message, '')
      return true
    },
  )
})

test('an error that is not the rule is passed through untouched', async () => {
  const run = async () => {
    throw new Error('Cannot reach the server.')
  }
  await assert.rejects(() => withConfirm(run, async () => {
    throw new Error('the dialog should not open')
  }), /Cannot reach the server\./)
})

test('concurrent 428s share one dialog', async () => {
  let opened = 0
  let release = null
  setConfirmAsker(
    () =>
      new Promise((resolve) => {
        opened += 1
        release = resolve
      }),
  )
  const first = askOnce()
  const second = askOnce()
  assert.equal(opened, 1)
  release({ until: new Date(Date.now() + 300000).toISOString() })
  await Promise.all([first, second])
  assert.equal(opened, 1)
  assert.ok(confirmedUntil() > Date.now())
  const third = askOnce()
  assert.equal(opened, 2)
  release({})
  await third
  setConfirmAsker(null)
  forgetConfirmed()
})

test('the confirmed window is remembered and forgotten', () => {
  forgetConfirmed()
  assert.equal(confirmedUntil(), 0)
  markConfirmed(Date.now() + 120000)
  assert.ok(confirmedUntil() > Date.now())
  markConfirmed(Date.now() - 1000)
  assert.equal(confirmedUntil(), 0)
  forgetConfirmed()
})

test('cancelling without a dialog is still quiet', async () => {
  setConfirmAsker(null)
  await assert.rejects(() => askOnce(), (failure) => isCancelled(failure))
})
