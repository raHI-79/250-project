import { useRef, useState } from 'react'
import { Paperclip } from 'lucide-react'
import { Button, Field } from '../ui'

const MAX_MB = 10

const MAX_BYTES = MAX_MB * 1024 * 1024

export default function PdfInput({ file, onChange, label = 'PDF attachment', className }) {
  const [problem, setProblem] = useState('')
  const control = useRef(null)
  const clearControl = () => {
    if (control.current) control.current.value = ''
  }
  const pick = (event) => {
    const chosen = event.target.files?.[0] || null
    if (!chosen) {
      setProblem('')
      onChange(null)
      return
    }
    if (chosen.type !== 'application/pdf' && !chosen.name.toLowerCase().endsWith('.pdf')) {
      setProblem('Only a PDF file can be attached.')
      onChange(null)
      clearControl()
      return
    }
    if (chosen.size > MAX_BYTES) {
      setProblem(`The PDF must be at most ${MAX_MB} MB.`)
      onChange(null)
      clearControl()
      return
    }
    setProblem('')
    onChange(chosen)
  }
  const remove = () => {
    setProblem('')
    onChange(null)
    clearControl()
  }
  return (
    <Field
      label={label}
      error={problem}
      hint={`Optional, PDF only, at most ${MAX_MB} MB`}
      className={className}
    >
      {file ? (
        <div className="flex items-center justify-between gap-3 rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm">
          <span className="inline-flex min-w-0 items-center gap-2 text-brand-800">
            <Paperclip size={15} className="shrink-0 text-brand-500" />
            <span className="truncate">{file.name}</span>
          </span>
          <Button variant="ghost" size="sm" onClick={remove}>
            Remove
          </Button>
        </div>
      ) : (
        <input
          ref={control}
          type="file"
          accept="application/pdf,.pdf"
          onChange={pick}
          className="w-full rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm text-brand-700 file:mr-3 file:rounded-md file:border-0 file:bg-brand-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-brand-700 hover:file:bg-brand-100"
        />
      )}
    </Field>
  )
}
