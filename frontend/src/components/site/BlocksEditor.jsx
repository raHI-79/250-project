import { useState } from 'react'
import { Plus } from 'lucide-react'
import { BLOCK_TYPES, move, newBlock } from '../../lib/siteContent'
import { Button, Input, Select } from '../ui'
import { ImageField, LinkFields, LongText, RowTools } from './EditorFields'

const labelOf = (type) => BLOCK_TYPES.find((entry) => entry.value === type)?.label || type

function ItemList({ items, onChange, empty, addLabel, render }) {
  return (
    <div className="flex flex-col gap-3">
      {items.map((item, index) => (
        <div key={`item-${index}`} className="rounded-xl border border-line bg-wash p-3">
          <div className="mb-2 flex justify-end">
            <RowTools
              index={index}
              total={items.length}
              label="this item"
              onMove={(step) => onChange(move(items, index, step))}
              onRemove={items.length > 1 ? () => onChange(items.filter((_, at) => at !== index)) : null}
            />
          </div>
          {render(item, (next) => onChange(items.map((entry, at) => (at === index ? next : entry))))}
        </div>
      ))}
      <div>
        <Button variant="secondary" size="sm" onClick={() => onChange([...items, { ...empty }])}>
          <Plus size={15} />
          {addLabel}
        </Button>
      </div>
    </div>
  )
}

function BlockFields({ block, onChange }) {
  const set = (patch) => onChange({ ...block, ...patch })
  if (block.type === 'heading') {
    return (
      <Input
        label="Heading"
        value={block.text || ''}
        maxLength={300}
        onChange={(event) => set({ text: event.target.value })}
      />
    )
  }
  if (block.type === 'text') {
    return <LongText label="Text" value={block.text} onChange={(text) => set({ text })} />
  }
  if (block.type === 'image' || block.type === 'image_text') {
    return (
      <div className="flex flex-col gap-3">
        <ImageField label="Image" value={block.image_url} onChange={(image_url) => set({ image_url })} />
        <Input
          label="What the image shows"
          value={block.alt || ''}
          maxLength={300}
          hint="Read aloud to people who cannot see the image."
          onChange={(event) => set({ alt: event.target.value })}
        />
        {block.type === 'image_text' && (
          <>
            <Select
              label="Image side"
              value={block.side || 'left'}
              onChange={(event) => set({ side: event.target.value })}
            >
              <option value="left">Image on the left</option>
              <option value="right">Image on the right</option>
            </Select>
            <LongText label="Text" value={block.text} onChange={(text) => set({ text })} />
          </>
        )}
      </div>
    )
  }
  if (block.type === 'button') {
    return (
      <LinkFields
        value={{ label: block.label || '', url: block.url || '' }}
        labelText="Button text"
        urlText="Button link"
        onChange={(next) => set({ label: next.label, url: next.url })}
      />
    )
  }
  if (block.type === 'cards') {
    return (
      <ItemList
        items={block.cards || []}
        empty={{ title: '', text: '', date: '', image_url: '' }}
        addLabel="Add a card"
        onChange={(cards) => set({ cards })}
        render={(card, change) => (
          <div className="flex flex-col gap-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <Input
                label="Card title"
                value={card.title || ''}
                maxLength={300}
                onChange={(event) => change({ ...card, title: event.target.value })}
              />
              <Input
                label="Date"
                type="date"
                value={card.date || ''}
                onChange={(event) => change({ ...card, date: event.target.value })}
              />
            </div>
            <LongText
              label="Card text"
              rows={3}
              value={card.text}
              onChange={(text) => change({ ...card, text })}
            />
            <ImageField
              label="Card image"
              value={card.image_url}
              onChange={(image_url) => change({ ...card, image_url })}
            />
          </div>
        )}
      />
    )
  }
  return (
    <ItemList
      items={block.faq || []}
      empty={{ question: '', answer: '' }}
      addLabel="Add a question"
      onChange={(faq) => set({ faq })}
      render={(entry, change) => (
        <div className="flex flex-col gap-3">
          <Input
            label="Question"
            value={entry.question || ''}
            maxLength={300}
            onChange={(event) => change({ ...entry, question: event.target.value })}
          />
          <LongText
            label="Answer"
            rows={3}
            value={entry.answer}
            onChange={(answer) => change({ ...entry, answer })}
          />
        </div>
      )}
    />
  )
}

export default function BlocksEditor({ blocks, onChange }) {
  const [type, setType] = useState(BLOCK_TYPES[0].value)
  const items = blocks || []
  return (
    <div data-blocks-editor className="flex flex-col gap-4">
      {items.length === 0 && <p className="text-sm text-muted">No blocks yet. Add the first one below.</p>}
      {items.map((block, index) => (
        <section
          key={`block-${index}`}
          data-block-editor={block.type}
          className="rounded-card border border-line bg-surface p-4"
        >
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <p className="label-caps">{labelOf(block.type)}</p>
            <RowTools
              index={index}
              total={items.length}
              label="this block"
              onMove={(step) => onChange(move(items, index, step))}
              onRemove={() => onChange(items.filter((_, at) => at !== index))}
            />
          </div>
          <BlockFields
            block={block}
            onChange={(next) => onChange(items.map((entry, at) => (at === index ? next : entry)))}
          />
        </section>
      ))}
      <div className="flex flex-wrap items-end gap-3">
        <Select
          label="Add a block"
          value={type}
          onChange={(event) => setType(event.target.value)}
          className="min-w-52"
        >
          {BLOCK_TYPES.map((entry) => (
            <option key={entry.value} value={entry.value}>
              {entry.label}
            </option>
          ))}
        </Select>
        <Button variant="secondary" onClick={() => onChange([...items, newBlock(type)])}>
          <Plus size={16} />
          Add block
        </Button>
      </div>
    </div>
  )
}
