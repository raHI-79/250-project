import { Select } from '../ui'
import { SECTIONS } from '../../lib/options'

export default function SectionPicker({
  value,
  onChange,
  error,
  className,
  emptyLabel = 'Choose a section',
  hint = 'The department runs section A and section B',
  required = true,
}) {
  return (
    <Select
      label="Section"
      value={value}
      onChange={(event) => onChange(event.target.value)}
      error={error}
      hint={hint}
      required={required}
      className={className}
    >
      <option value="">{emptyLabel}</option>
      {SECTIONS.map((section) => (
        <option key={section} value={section}>
          Section {section}
        </option>
      ))}
    </Select>
  )
}
