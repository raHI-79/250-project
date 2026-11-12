export const EMAIL_CODE_MINUTES = 15

export const RESEND_SECONDS = 60

export const CODE_CONTACT_TEXT =
  'We email a 6-digit code to your university address. Tap Send code, enter the code, then continue.'

export const GOOGLE_CONTACT_TEXT =
  'Verify your university address with Google, or let us email you a code.'

export const EXPIRED_TEXT = 'This code has expired. Tap Resend the code to get a new one.'

export const googleOffered = ({ app, enabled } = {}) => !app && Boolean(enabled)

export const contactText = (offered) => (offered ? GOOGLE_CONTACT_TEXT : CODE_CONTACT_TEXT)

export const codeDeadline = (expiresAt, sentAt) => {
  const stated = expiresAt ? new Date(expiresAt) : null
  if (stated && !Number.isNaN(stated.getTime())) return stated
  if (!sentAt) return null
  return new Date(new Date(sentAt).getTime() + EMAIL_CODE_MINUTES * 60000)
}

export const codeExpired = (deadline, now) =>
  Boolean(deadline) && now.getTime() >= deadline.getTime()

export const codeHint = ({ emailHint, deadline, now }) => {
  const where = emailHint ? `Sent to ${emailHint}.` : 'Sent to your university email address.'
  if (!deadline) return `${where} It expires in ${EMAIL_CODE_MINUTES} minutes.`
  if (codeExpired(deadline, now)) return `${where} This code has expired.`
  const left = Math.max(1, Math.ceil((deadline.getTime() - now.getTime()) / 60000))
  return `${where} It expires in ${left} minute${left === 1 ? '' : 's'}.`
}

export const resendLabel = (seconds) => (seconds > 0 ? `Resend in ${seconds}s` : 'Resend the code')

export const resendNote = (seconds) =>
  seconds > 0
    ? 'Not in your inbox? Check the spam folder. You can ask for a new code when the timer ends.'
    : 'Not in your inbox? Check the spam folder, then ask for a new code.'

export const resentText = (emailHint) =>
  `A new code was sent${emailHint ? ` to ${emailHint}` : ''}. The earlier code no longer works.`
