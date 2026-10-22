import { useState } from 'react'
import { Link2, Paperclip, Send } from 'lucide-react'
import FileInput from '../forms/FileInput'
import LinkInput from '../forms/LinkInput'
import { Alert, Button, Textarea } from '../ui'
import { linkError } from '../../lib/files'
import { MAX_MESSAGE_BODY } from '../../lib/messages'

export default function MessageComposer({ onSend }) {
  const [body, setBody] = useState('')
  const [link, setLink] = useState('')
  const [file, setFile] = useState(null)
  const [showLink, setShowLink] = useState(false)
  const [showFile, setShowFile] = useState(false)
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
      setShowLink(false)
      setShowFile(false)
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
  return (
    <div className="flex flex-col gap-3 border-t border-line p-3">
      {error && <Alert tone="error">{error}</Alert>}
      {showLink && <LinkInput value={link} onChange={setLink} />}
      {showFile && <FileInput file={file} onChange={setFile} />}
      <Textarea
        value={body}
        onChange={(event) => setBody(event.target.value)}
        onKeyDown={onKeyDown}
        rows={2}
        maxLength={MAX_MESSAGE_BODY}
        placeholder="Write a message. Enter sends it, Shift and Enter make a new line."
        aria-label="Message"
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant={showLink ? 'secondary' : 'ghost'}
          size="sm"
          onClick={() => setShowLink((current) => !current)}
        >
          <Link2 size={15} />
          Add a link
        </Button>
        <Button
          variant={showFile ? 'secondary' : 'ghost'}
          size="sm"
          onClick={() => setShowFile((current) => !current)}
        >
          <Paperclip size={15} />
          Attach a file
        </Button>
        <Button loading={sending} onClick={submit} className="ml-auto">
          <Send size={15} />
          Send
        </Button>
      </div>
    </div>
  )
}
