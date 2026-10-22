import { Field, Textarea } from '../ui'
import { MAX_NOTE, NOTE_AUDIENCES } from '../../lib/options'

export default function NoteFields({ note, audience, onNoteChange, onAudienceChange, error }) {
  return (
    <>
      <Textarea
        label="Additional info"
        value={note}
        onChange={(event) => onNoteChange(event.target.value)}
        rows={3}
        maxLength={MAX_NOTE}
        placeholder="Anything the teacher or the students should know"
        hint={`Optional, ${note.length} of ${MAX_NOTE} characters`}
      />
      {note.trim() !== '' && (
        <Field label="Who should read it?" error={error} required className="animate-fade-in">
          <div className="flex flex-wrap gap-4">
            {NOTE_AUDIENCES.map((entry) => (
              <label
                key={entry.value}
                className="inline-flex items-center gap-2 text-sm text-ink"
              >
                <input
                  type="radio"
                  name="note-audience"
                  value={entry.value}
                  checked={audience === entry.value}
                  onChange={() => onAudienceChange(entry.value)}
                  className="size-4 accent-brand-600"
                />
                {entry.label}
              </label>
            ))}
          </div>
        </Field>
      )}
    </>
  )
}
