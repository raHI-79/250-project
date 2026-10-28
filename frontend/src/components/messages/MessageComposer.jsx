import { useState } from 'react'
import { Link2, Paperclip, SendHorizontal, X } from 'lucide-react'
import FileInput from '../forms/FileInput'
import LinkInput from '../forms/LinkInput'
import { Alert, Button, Sheet } from '../ui'
import { linkError } from '../../lib/files'
import { MAX_MESSAGE_BODY } from '../../lib/messages'

export default function MessageComposer({ onSend }) {
  const [body, setBody] = useState('')
  const [link, setLink] = useState('')
  const [file, setFile] = useState(null)
  const [attachOpen, setAttachOpen] = useState(false)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const submit = async () => {
    if (sending) return
    const text = body.trim()
    const url = link.trim()
    if (!text && !url && !file) {
      setError('Write a message, add a link or attach a file.')
      return
    }
    const problem = linkError(url)
    if (problem) {
      setError(problem)
      return
    }
    setSending(true)
    try {
      await onSend({ body: text, link: url, file })
      setBody('')
      setLink('')
      setFile(null)
      setError('')
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const onKeyDown = (event) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      submit()
    }
  }
  const attached = Boolean(file) || Boolean(link.trim())
  return (
    <div className="glass sticky bottom-0 border-t px-3 pt-2 pb-[max(0.5rem,env(safe-area-inset-bottom,0px))]">
      {error && (
        <Alert tone="error" className="mb-2">
          {error}
        </Alert>
      )}
      {attached && (
        <div className="mb-2 flex items-center gap-2 rounded-xl bg-wash px-3 py-2 text-xs text-ink-soft">
          <Paperclip size={14} className="shrink-0 text-brand-600" />
          <span className="min-w-0 flex-1 truncate">{file ? file.name : link.trim()}</span>
          <button
            type="button"
            onClick={() => {
              setFile(null)
              setLink('')
            }}
            aria-label="Remove the attachment"
            className="shrink-0 text-faint hover:text-ink"
          >
            <X size={14} />
          </button>
        </div>
      )}
      <div className="flex items-end gap-2">
        <button
          type="button"
          onClick={() => setAttachOpen(true)}
          aria-label="Add an attachment"
          className="press glass-chip grid size-11 shrink-0 place-items-center rounded-full text-brand-600"
        >
          <Paperclip size={19} />
        </button>
        <textarea
          value={body}
          onChange={(event) => setBody(event.target.value)}
          onKeyDown={onKeyDown}
          rows={1}
          maxLength={MAX_MESSAGE_BODY}
          placeholder="Message"
          aria-label="Message"
          className="glass-chip max-h-32 min-h-11 flex-1 resize-none rounded-2xl px-4 py-2.5 text-base text-ink placeholder:text-faint focus:border-brand-600 focus:ring-3 focus:ring-brand-100 focus:outline-none"
        />
        <button
          type="button"
          onClick={submit}
          disabled={sending}
          aria-label="Send"
          className="press grid size-11 shrink-0 place-items-center rounded-full bg-linear-to-b from-brand-500 to-brand-700 text-page shadow-[var(--shadow-glass)] hover:from-brand-600 hover:to-brand-800 disabled:from-brand-300 disabled:to-brand-300"
        >
          <SendHorizontal size={19} />
        </button>
      </div>
      <Sheet open={attachOpen} onClose={() => setAttachOpen(false)} title="Attach">
        <div className="flex flex-col gap-4">
          <FileInput file={file} onChange={setFile} />
          <LinkInput value={link} onChange={setLink} />
          <Button onClick={() => setAttachOpen(false)}>
            <Link2 size={16} />
            Done
          </Button>
        </div>
      </Sheet>
    </div>
  )
}
