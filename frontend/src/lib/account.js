export const MIN_AGE = 15

export const MAX_AGE = 60

export const MAX_FIELD = 120

export const MAX_BIO = 500

export const BATCH_PATTERN = /^[A-Z]{2,6}-[0-9]{2}$/

export const SECTION_PATTERN = /^[A-Z]$/

export const CODE_PATTERN = /^[0-9]{6}$/

export const PHONE_PATTERN = /^01[3-9][0-9]{8}$/

export const PHONE_MESSAGE = 'Phone number must be 11 digits, for example 01712345678.'

export const isPhone = (value) => PHONE_PATTERN.test(String(value || '').trim())

export const initialsOf = (name) => {
  const parts = String(name || '')
    .trim()
    .split(/\s+/)
    .filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

export const BIRTH_DATE_MESSAGE = `Choose a date of birth between ${MIN_AGE} and ${MAX_AGE} years ago.`

const isoDate = (date) =>
  [
    date.getFullYear(),
    String(date.getMonth() + 1).padStart(2, '0'),
    String(date.getDate()).padStart(2, '0'),
  ].join('-')

export const birthDateBounds = () => {
  const today = new Date()
  const youngest = new Date(today.getFullYear() - MIN_AGE, today.getMonth(), today.getDate())
  const oldest = new Date(today.getFullYear() - MAX_AGE - 1, today.getMonth(), today.getDate() + 1)
  return { min: isoDate(oldest), max: isoDate(youngest) }
}

export const birthDateError = (value) => {
  const text = String(value || '').trim()
  if (!text) return 'Date of birth is required.'
  if (!/^[0-9]{4}-[0-9]{2}-[0-9]{2}$/.test(text)) return BIRTH_DATE_MESSAGE
  const bounds = birthDateBounds()
  if (text < bounds.min || text > bounds.max) return BIRTH_DATE_MESSAGE
  return ''
}

export const emailDomainError = (value, domain) => {
  const text = String(value || '')
    .trim()
    .toLowerCase()
  if (!text) return 'Email is required.'
  if (!/^[^@\s]+@[^@\s]+\.[^@\s.]{2,}$/.test(text)) return 'Email must be a valid email address.'
  if (domain && !text.endsWith(`@${domain.toLowerCase()}`)) return `Email must end with @${domain}.`
  return ''
}

export const passwordError = (password, confirm) => {
  if (!password) return 'Password is required.'
  if (password.length < 8) return 'Password must be at least 8 characters long.'
  if (password.length > 72) return 'Password must be at most 72 characters long.'
  if (confirm !== undefined && password !== confirm)
    return 'Password and the confirmation do not match.'
  return ''
}

export const accountRoleLabel = (role) => {
  if (role === 'admin') return 'Administrator'
  if (role === 'teacher') return 'Faculty'
  if (role === 'cr') return 'Class Representative'
  return 'Student'
}

export const appendIf = (form, key, value) => {
  const text = value === undefined || value === null ? '' : String(value)
  if (text) form.append(key, text)
}
