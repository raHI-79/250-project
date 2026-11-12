import { useRef, useState } from 'react'
import { ArrowDown, ArrowUp, ImagePlus, Plus, Trash2 } from 'lucide-react'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { assetURL } from '../../lib/native'
import { isSafeImage, isSafeLink, move } from '../../lib/siteContent'
import { Button, Input, Textarea, useToast } from '../ui'

const icon =
  'press grid size-10 shrink-0 place-items-center rounded-xl border border-line-strong bg-surface text-muted hover:text-ink disabled:opacity-40'

export function RowTools({ index, total, onMove, onRemove, label }) {
  return (
    <span className="flex shrink-0 items-center gap-1.5">
      <button
        type="button"
        aria-label={`Move ${label} up`}
        disabled={index === 0}
        onClick={() => onMove(-1)}
        className={icon}
      >
        <ArrowUp size={16} />
      </button>
      <button
        type="button"
        aria-label={`Move ${label} down`}
        disabled={index === total - 1}
        onClick={() => onMove(1)}
        className={icon}
      >
        <ArrowDown size={16} />
      </button>
      {onRemove && (
        <button
          type="button"
          aria-label={`Remove ${label}`}
          onClick={onRemove}
          className={cx(icon, 'text-danger-600 hover:text-danger-600')}
        >
          <Trash2 size={16} />
        </button>
      )}
    </span>
  )
}

export function LongText({ label, value, onChange, hint, rows = 4, maxLength = 4000 }) {
  return (
    <Textarea
      label={label}
      value={value || ''}
      rows={rows}
      maxLength={maxLength}
      onChange={(event) => onChange(event.target.value)}
      hint={hint || 'Use **bold**, *italic*, lines starting with - for a list and [text](https://…) for a link.'}
    />
  )
}

export function ImageField({ label, value, onChange, hint }) {
  const toast = useToast()
  const picker = useRef(null)
  const [busy, setBusy] = useState(false)
  const upload = async (event) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    const body = new FormData()
    body.append('image', file)
    setBusy(true)
    try {
      const saved = await api.post('/api/super-admin/site/images', body)
      onChange(saved.url)
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <div data-image-field={label}>
      <p className="text-sm font-semibold text-ink">{label}</p>
      <div className="mt-2 flex flex-wrap items-center gap-3">
        {isSafeImage(value) ? (
          <img
            src={assetURL(value)}
            alt=""
            className="h-16 w-24 rounded-xl border border-line bg-wash object-cover"
          />
        ) : (
          <span className="grid h-16 w-24 place-items-center rounded-xl border border-dashed border-line-strong text-xs text-faint">
            No image
          </span>
        )}
        <input
          ref={picker}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          onChange={upload}
          className="sr-only"
          aria-label={`Upload ${label}`}
        />
        <Button variant="secondary" size="sm" loading={busy} onClick={() => picker.current?.click()}>
          <ImagePlus size={15} />
          {value ? 'Replace' : 'Upload'}
        </Button>
        {value && (
          <Button variant="ghost" size="sm" onClick={() => onChange('')}>
            Remove
          </Button>
        )}
      </div>
      <p className="mt-1 text-xs text-muted">{hint || 'JPG, PNG or WEBP, up to 4 MB.'}</p>
    </div>
  )
}

export function LinkFields({ value, onChange, labelText = 'Text', urlText = 'Link' }) {
  const link = value || { label: '', url: '' }
  const bad = link.url !== '' && !isSafeLink(link.url)
  return (
    <div className="grid min-w-0 flex-1 gap-3 sm:grid-cols-2">
      <Input
        label={labelText}
        value={link.label}
        maxLength={120}
        onChange={(event) => onChange({ ...link, label: event.target.value })}
      />
      <Input
        label={urlText}
        value={link.url}
        maxLength={300}
        placeholder="https://… or /p/page"
        error={bad ? 'Use https://… or a page of this site like /p/admission.' : ''}
        onChange={(event) => onChange({ ...link, url: event.target.value })}
      />
    </div>
  )
}

export function LinksEditor({ title, links, onChange, addLabel = 'Add a link' }) {
  const items = links || []
  return (
    <div data-links-editor={title}>
      <p className="text-sm font-semibold text-ink">{title}</p>
      <ul className="mt-2 flex flex-col gap-3">
        {items.map((entry, index) => (
          <li key={`link-${index}`} className="flex flex-wrap items-end gap-3">
            <LinkFields
              value={entry}
              onChange={(next) => onChange(items.map((item, at) => (at === index ? next : item)))}
            />
            <RowTools
              index={index}
              total={items.length}
              label="this link"
              onMove={(step) => onChange(move(items, index, step))}
              onRemove={() => onChange(items.filter((_, at) => at !== index))}
            />
          </li>
        ))}
      </ul>
      <Button
        variant="secondary"
        size="sm"
        className="mt-3"
        onClick={() => onChange([...items, { label: '', url: '' }])}
      >
        <Plus size={15} />
        {addLabel}
      </Button>
    </div>
  )
}
