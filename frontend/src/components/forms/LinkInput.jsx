import { useState } from 'react'
import { Input } from '../ui'
import { linkError } from '../../lib/files'

export default function LinkInput({ value, onChange, label = 'Link', hint, className }) {
  const [touched, setTouched] = useState(false)
  const problem = touched ? linkError(value) : ''
  return (
    <Input
      label={label}
      type="url"
      value={value}
      onChange={(event) => onChange(event.target.value)}
      onBlur={() => setTouched(true)}
      placeholder="https://drive.google.com/file/..."
      error={problem}
      hint={hint || 'Optional, a full http:// or https:// address'}
      className={className}
    />
  )
}
