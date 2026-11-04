import assert from 'node:assert/strict'
import test from 'node:test'
import { registrationErrors, resetErrors } from './account.js'

const DOMAIN = 'student.sust.edu'

const complete = {
  fullName: 'Ratul Islam',
  regNo: '2023331079',
  dateOfBirth: '2003-05-14',
  batch: 'CSE-23',
  section: 'A',
  email: 'ratul@student.sust.edu',
  phone: '01712345678',
  bloodGroup: 'O+',
  password: 'a-good-password',
  confirmPassword: 'a-good-password',
}

test('a complete registration form has nothing to correct', () => {
  assert.deepEqual(registrationErrors(complete, DOMAIN), {})
})

test('the single page names every field that is wrong at once', () => {
  const found = registrationErrors(
    { ...complete, fullName: '', batch: 'CSE23', section: 'AB', email: 'ratul@gmail.com', phone: '12' },
    DOMAIN,
  )
  assert.deepEqual(Object.keys(found).sort(), ['batch', 'email', 'fullName', 'phone', 'section'])
  assert.match(found.email, /@student\.sust\.edu/)
  assert.match(found.batch, /CSE-23/)
})

test('the password and its confirmation have to match', () => {
  const found = registrationErrors({ ...complete, confirmPassword: 'another' }, DOMAIN)
  assert.match(found.password, /do not match/)
})

test('a short password is refused', () => {
  const found = registrationErrors({ ...complete, password: 'short', confirmPassword: 'short' }, DOMAIN)
  assert.match(found.password, /at least 8/)
})

test('a date of birth outside the window is refused', () => {
  const found = registrationErrors({ ...complete, dateOfBirth: '2024-01-01' }, DOMAIN)
  assert.match(found.dateOfBirth, /between 15 and 60/)
})

test('the email code is checked when the form carries one', () => {
  assert.deepEqual(registrationErrors(complete, DOMAIN, { code: '123456' }), {})
  const found = registrationErrors(complete, DOMAIN, { code: '12' })
  assert.match(found.code, /6-digit code/)
})

test('the reset asks for the account first', () => {
  const found = resetErrors({ identifier: '', dateOfBirth: '' }, { stage: 'identify' })
  assert.deepEqual(Object.keys(found).sort(), ['dateOfBirth', 'identifier'])
})

test('the administration reset checks the phone instead of the date', () => {
  const found = resetErrors({ identifier: 'office', phone: '12' }, { byPhone: true, stage: 'identify' })
  assert.deepEqual(Object.keys(found), ['phone'])
  assert.deepEqual(
    resetErrors({ identifier: 'office', phone: '01712345678' }, { byPhone: true, stage: 'identify' }),
    {},
  )
})

test('the whole reset page checks the code and the new password together', () => {
  const found = resetErrors({
    identifier: '2023331079',
    dateOfBirth: '2003-05-14',
    code: '12',
    password: 'a-good-password',
    confirmPassword: 'another-password',
  })
  assert.deepEqual(Object.keys(found).sort(), ['code', 'password'])
  assert.match(found.password, /New password/)
})

test('a filled reset page has nothing to correct', () => {
  assert.deepEqual(
    resetErrors({
      identifier: '2023331079',
      dateOfBirth: '2003-05-14',
      code: '123456',
      password: 'a-good-password',
      confirmPassword: 'a-good-password',
    }),
    {},
  )
})
