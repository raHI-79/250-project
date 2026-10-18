import { useEffect, useRef, useState } from 'react'
import { ImagePlus } from 'lucide-react'
import { Button, Field } from '../ui'
import cx from '../../lib/cx'

export const PHOTO_MAX_MB = 2

export const PHOTO_ACCEPT = 'image/jpeg,image/png,image/webp'

const MAX_BYTES = PHOTO_MAX_MB * 1024 * 1024

const TYPES = ['image/jpeg', 'image/png', 'image/webp']

const NOT_IMAGE = 'Choose a JPG, PNG or WEBP image.'

const TOO_LARGE = `The photo must be ${PHOTO_MAX_MB} MB or smaller.`

export default function PhotoPicker({
  file,
  onChange,
  currentUrl = '',
  label = 'Profile photo',
  hint = `JPG, PNG or WEBP, at most ${PHOTO_MAX_MB} MB, at least 200 by 200 pixels`,
  error = '',
  required = false,
  className,
}) {
  const [problem, setProblem] = useState('')
  const [preview, setPreview] = useState('')
  const [dragging, setDragging] = useState(false)
  const control = useRef(null)
  useEffect(() => {
    if (!file) {
      setPreview('')
      return undefined
    }
    const url = URL.createObjectURL(file)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [file])
  const clearControl = () => {
    if (control.current) control.current.value = ''
  }
  const accept = (chosen) => {
    if (!chosen) return
    if (!TYPES.includes(chosen.type)) {
      setProblem(NOT_IMAGE)
      onChange(null)
      clearControl()
      return
    }
    if (chosen.size > MAX_BYTES) {
      setProblem(TOO_LARGE)
      onChange(null)
      clearControl()
      return
    }
    setProblem('')
    onChange(chosen)
  }
  const onPick = (event) => accept(event.target.files?.[0] || null)
  const onDrop = (event) => {
    event.preventDefault()
    setDragging(false)
    accept(event.dataTransfer?.files?.[0] || null)
  }
  const remove = () => {
    setProblem('')
    onChange(null)
    clearControl()
  }
  const shown = preview || currentUrl
  return (
    <Field label={label} error={error || problem} hint={hint} required={required} className={className}>
      <div className="flex items-start gap-4">
        <div
          onDragOver={(event) => {
            event.preventDefault()
            setDragging(true)
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={onDrop}
          className={cx(
            'relative size-28 shrink-0 overflow-hidden rounded-xl border-2 border-dashed transition-colors',
            dragging ? 'border-brand-500 bg-brand-50' : 'border-brand-200 bg-brand-50/50',
          )}
        >
          {shown ? (
            <img src={shown} alt="Profile photo preview" className="size-full object-cover" />
          ) : (
            <span className="flex size-full flex-col items-center justify-center gap-1 text-brand-400">
              <ImagePlus size={22} />
              <span className="px-2 text-center text-[11px] leading-tight">Click or drop</span>
            </span>
          )}
          <button
            type="button"
            onClick={() => control.current?.click()}
            aria-label={shown ? 'Change the photo' : 'Choose a photo'}
            className="absolute inset-0 cursor-pointer"
          />
        </div>
        <div className="flex min-w-0 flex-col gap-2">
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" size="sm" onClick={() => control.current?.click()}>
              {shown ? 'Change photo' : 'Choose photo'}
            </Button>
            {file && (
              <Button variant="ghost" size="sm" onClick={remove}>
                Remove
              </Button>
            )}
          </div>
          {file && <p className="truncate text-xs text-brand-500">{file.name}</p>}
        </div>
      </div>
      <input
        ref={control}
        type="file"
        accept={PHOTO_ACCEPT}
        onChange={onPick}
        className="hidden"
      />
    </Field>
  )
}
