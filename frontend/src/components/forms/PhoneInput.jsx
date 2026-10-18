import { useState } from 'react'
import { Input } from '../ui'
import { PHONE_MESSAGE, isPhone } from '../../lib/account'

const digitsOnly = (value) => String(value || '').replace(/[^0-9]/g, '').slice(0, 11)

export default function PhoneInput({
  value,
  onChange,
  label = 'Phone',
  error = '',
  hint = 'For example 01712345678',
  required = false,
  className,
  ...rest
}) {
  const [touched, setTouched] = useState(false)
  const live = touched && value && !isPhone(value) ? PHONE_MESSAGE : ''
  return (
    <Input
      label={label}
      value={value}
      onChange={(event) => onChange(digitsOnly(event.target.value))}
      onBlur={() => setTouched(true)}
      error={error || live}
      hint={hint}
      required={required}
      className={className}
      inputMode="numeric"
      autoComplete="tel"
      maxLength={11}
      {...rest}
    />
  )
}
