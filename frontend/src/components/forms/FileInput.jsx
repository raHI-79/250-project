import { useRef, useState } from 'react'
import { Paperclip } from 'lucide-react'
import { Button, Field } from '../ui'
import { FILE_ACCEPT, MAX_FILE_MB, fileError, formatFileSize } from '../../lib/files'

export default function FileInput({ file, onChange, label = 'Attachment', hint, className }) {
  const [problem, setProblem] = useState('')
  const control = useRef(null)
  const clearControl = () => {
    if (control.current) control.current.value = ''
  }
  const pick = (event) => {
    const chosen = event.target.files?.[0] || null
    const failure = fileError(chosen)
    if (failure) {
      setProblem(failure)
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
      hint={hint || `Optional, at most ${MAX_FILE_MB} MB`}
      className={className}
    >
      {file ? (
        <div className="flex items-center justify-between gap-3 rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm">
          <span className="inline-flex min-w-0 items-center gap-2 text-brand-800">
            <Paperclip size={15} className="shrink-0 text-brand-500" />
            <span className="truncate">{file.name}</span>
            <span className="shrink-0 text-xs text-brand-400">{formatFileSize(file.size)}</span>
          </span>
          <Button variant="ghost" size="sm" onClick={remove}>
            Remove
          </Button>
        </div>
      ) : (
        <input
          ref={control}
          type="file"
          accept={FILE_ACCEPT}
          onChange={pick}
          className="w-full rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm text-brand-700 file:mr-3 file:rounded-md file:border-0 file:bg-brand-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-brand-700 hover:file:bg-brand-100"
        />
      )}
    </Field>
  )
}
