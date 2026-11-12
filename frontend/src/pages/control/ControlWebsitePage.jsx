import { useEffect, useState } from 'react'
import { Eye, Plus, Save, UploadCloud } from 'lucide-react'
import BlocksEditor from '../../components/site/BlocksEditor'
import {
  ImageField,
  LinkFields,
  LinksEditor,
  LongText,
  RowTools,
} from '../../components/site/EditorFields'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  Input,
  PageHeader,
  SkeletonCard,
  StatusTabs,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import paths from '../../lib/paths'
import {
  CUSTOM_PREFIX,
  DEVELOPER,
  homeSections,
  move,
  newId,
  slugFrom,
  slugProblem,
  stripCredit,
  withDefaults,
} from '../../lib/siteContent'
import { formatDateTime } from '../../lib/time'
import { forgetSiteContent } from '../../lib/useSiteContent'

const TABS = [
  { value: 'header', label: 'Header' },
  { value: 'home', label: 'Home' },
  { value: 'sections', label: 'Sections' },
  { value: 'footer', label: 'Footer' },
  { value: 'more', label: 'Sign-in and SEO' },
  { value: 'pages', label: 'Pages' },
  { value: 'versions', label: 'Versions' },
]

function Toggle({ label, checked, onChange }) {
  return (
    <label className="flex min-h-11 items-center gap-2.5 text-sm text-ink">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="size-4 shrink-0 accent-brand-600"
      />
      {label}
    </label>
  )
}

function HeaderTab({ content, set }) {
  const header = content.header
  const change = (patch) => set({ ...content, header: { ...header, ...patch } })
  return (
    <Card hover={false}>
      <CardHeader title="Header" subtitle="The name, the tagline, the logo and the extra menu links." />
      <div className="mt-4 flex flex-col gap-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <Input
            label="Site name"
            value={header.site_name}
            maxLength={120}
            onChange={(event) => change({ site_name: event.target.value })}
          />
          <Input
            label="Tagline"
            value={header.tagline}
            maxLength={300}
            onChange={(event) => change({ tagline: event.target.value })}
          />
        </div>
        <ImageField
          label="Logo"
          value={header.logo_url}
          hint="Leave it empty to use the SUST logo."
          onChange={(logo_url) => change({ logo_url })}
        />
        <LinksEditor
          title="Extra links in the header menu"
          links={header.menu}
          onChange={(menu) => change({ menu })}
        />
      </div>
    </Card>
  )
}

function HomeTab({ content, set }) {
  const home = content.home
  const change = (patch) => set({ ...content, home: { ...home, ...patch } })
  return (
    <div className="flex flex-col gap-4">
      <Card hover={false}>
        <CardHeader title="Hero" subtitle="The first thing a visitor reads." />
        <div className="mt-4 flex flex-col gap-4">
          <Input
            label="Small line above the title"
            value={home.hero_eyebrow}
            maxLength={120}
            onChange={(event) => change({ hero_eyebrow: event.target.value })}
          />
          <Input
            label="Hero title"
            value={home.hero_title}
            maxLength={300}
            onChange={(event) => change({ hero_title: event.target.value })}
          />
          <LongText
            label="Hero text"
            rows={3}
            hint="Plain text."
            value={home.hero_text}
            onChange={(hero_text) => change({ hero_text })}
          />
          <div>
            <p className="text-sm font-semibold text-ink">First button</p>
            <div className="mt-2">
              <LinkFields value={home.primary_cta} onChange={(primary_cta) => change({ primary_cta })} />
            </div>
          </div>
          <div>
            <p className="text-sm font-semibold text-ink">Second button</p>
            <div className="mt-2">
              <LinkFields
                value={home.secondary_cta}
                onChange={(secondary_cta) => change({ secondary_cta })}
              />
            </div>
          </div>
        </div>
      </Card>
      <Card hover={false}>
        <CardHeader title="Campus slideshow" subtitle="Upload, reorder, caption or remove the photos." />
        <ul className="mt-4 flex flex-col gap-4">
          {home.slides.map((slide, index) => (
            <li key={`slide-${index}`} data-slide className="rounded-xl border border-line bg-wash p-3">
              <div className="flex flex-wrap items-end gap-3">
                <ImageField
                  label={`Photo ${index + 1}`}
                  value={slide.image_url}
                  onChange={(image_url) =>
                    change({
                      slides: home.slides.map((entry, at) =>
                        at === index ? { ...entry, image_url } : entry,
                      ),
                    })
                  }
                />
                <RowTools
                  index={index}
                  total={home.slides.length}
                  label="this photo"
                  onMove={(step) => change({ slides: move(home.slides, index, step) })}
                  onRemove={
                    home.slides.length > 1
                      ? () => change({ slides: home.slides.filter((_, at) => at !== index) })
                      : null
                  }
                />
              </div>
              <Input
                label="Caption"
                className="mt-3"
                value={slide.caption}
                maxLength={300}
                onChange={(event) =>
                  change({
                    slides: home.slides.map((entry, at) =>
                      at === index ? { ...entry, caption: event.target.value } : entry,
                    ),
                  })
                }
              />
            </li>
          ))}
        </ul>
        <Button
          variant="secondary"
          size="sm"
          className="mt-3"
          disabled={home.slides.length >= 10}
          onClick={() => change({ slides: [...home.slides, { image_url: '', caption: '' }] })}
        >
          <Plus size={15} />
          Add a photo
        </Button>
      </Card>
      <Card hover={false}>
        <CardHeader title="About the department" subtitle="Shown in the About section of the home page." />
        <div className="mt-4 flex flex-col gap-4">
          <Input
            label="About title"
            value={home.about_title}
            maxLength={300}
            onChange={(event) => change({ about_title: event.target.value })}
          />
          <LongText label="About text" value={home.about_text} onChange={(about_text) => change({ about_text })} />
        </div>
      </Card>
    </div>
  )
}

function SectionsTab({ content, set }) {
  const confirm = useConfirm()
  const home = content.home
  const [title, setTitle] = useState('')
  const [open, setOpen] = useState('')
  const sections = homeSections(content)
  const change = (patch) => set({ ...content, home: { ...home, ...patch } })
  const refs = sections.map((entry) => ({ key: entry.key, visible: entry.visible }))
  const add = () => {
    const id = newId(home.custom.map((entry) => entry.id))
    change({
      custom: [...home.custom, { id, title: title.trim(), blocks: [] }],
      sections: [...refs, { key: `${CUSTOM_PREFIX}${id}`, visible: true }],
    })
    setOpen(id)
    setTitle('')
  }
  const remove = async (section) => {
    const ok = await confirm({
      title: `Delete the section “${section.title}”?`,
      message: 'Its blocks are removed from the draft. Published versions keep their copy.',
      confirmLabel: 'Delete',
    })
    if (!ok) return
    change({
      custom: home.custom.filter((entry) => entry.id !== section.id),
      sections: refs.filter((entry) => entry.key !== `${CUSTOM_PREFIX}${section.id}`),
    })
  }
  return (
    <div className="flex flex-col gap-4">
      <Card hover={false}>
        <CardHeader
          title="What the home page shows, and in which order"
          subtitle="The hero is always first. Switch a section off to hide it."
        />
        <ul className="mt-4 flex flex-col gap-2">
          {sections.map((entry, index) => (
            <li
              key={entry.key}
              data-section-row={entry.key}
              className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line bg-wash px-3 py-2"
            >
              <Toggle
                label={entry.label}
                checked={entry.visible}
                onChange={(visible) =>
                  change({
                    sections: refs.map((ref) => (ref.key === entry.key ? { ...ref, visible } : ref)),
                  })
                }
              />
              <RowTools
                index={index}
                total={sections.length}
                label={entry.label}
                onMove={(step) => change({ sections: move(refs, index, step) })}
              />
            </li>
          ))}
        </ul>
      </Card>
      <Card hover={false}>
        <CardHeader
          title="Your own sections"
          subtitle="Build a section from blocks, then place it anywhere in the order above."
        />
        <div className="mt-4 flex flex-wrap items-end gap-3">
          <Input
            label="Title of a new section"
            value={title}
            maxLength={300}
            onChange={(event) => setTitle(event.target.value)}
            className="min-w-56 flex-1"
          />
          <Button variant="secondary" disabled={title.trim() === ''} onClick={add}>
            <Plus size={16} />
            Add section
          </Button>
        </div>
        <div className="mt-4 flex flex-col gap-4">
          {home.custom.map((section) => (
            <section
              key={section.id}
              data-custom-section={section.id}
              className="rounded-card border border-line p-4"
            >
              <div className="flex flex-wrap items-end gap-3">
                <Input
                  label="Section title"
                  value={section.title}
                  maxLength={300}
                  className="min-w-56 flex-1"
                  onChange={(event) =>
                    change({
                      custom: home.custom.map((entry) =>
                        entry.id === section.id ? { ...entry, title: event.target.value } : entry,
                      ),
                    })
                  }
                />
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setOpen(open === section.id ? '' : section.id)}
                >
                  {open === section.id ? 'Hide blocks' : `Edit blocks (${section.blocks.length})`}
                </Button>
                <Button variant="danger" size="sm" onClick={() => remove(section)}>
                  Delete
                </Button>
              </div>
              {open === section.id && (
                <div className="mt-4">
                  <BlocksEditor
                    blocks={section.blocks}
                    onChange={(blocks) =>
                      change({
                        custom: home.custom.map((entry) =>
                          entry.id === section.id ? { ...entry, blocks } : entry,
                        ),
                      })
                    }
                  />
                </div>
              )}
            </section>
          ))}
        </div>
      </Card>
    </div>
  )
}

function FooterTab({ content, set }) {
  const footer = content.footer
  const change = (patch) => set({ ...content, footer: { ...footer, ...patch } })
  return (
    <Card hover={false}>
      <CardHeader title="Footer" subtitle="Address, contact, office hours and the footer links." />
      <div className="mt-4 flex flex-col gap-4">
        <Toggle
          label="Show the footer details"
          checked={footer.visible !== false}
          onChange={(visible) => change({ visible })}
        />
        <LongText
          label="Address"
          rows={3}
          hint="One line per row."
          maxLength={300}
          value={footer.address}
          onChange={(address) => change({ address })}
        />
        <div className="grid gap-3 sm:grid-cols-3">
          <Input
            label="Phone"
            value={footer.phone}
            maxLength={120}
            onChange={(event) => change({ phone: event.target.value })}
          />
          <Input
            label="Email"
            value={footer.email}
            maxLength={120}
            onChange={(event) => change({ email: event.target.value })}
          />
          <Input
            label="Office hours"
            value={footer.office_hours}
            maxLength={300}
            onChange={(event) => change({ office_hours: event.target.value })}
          />
        </div>
        <Input
          label="Footer text"
          value={footer.note}
          maxLength={300}
          onChange={(event) => change({ note: event.target.value })}
        />
        <LinksEditor
          title="Quick links"
          links={footer.quick_links}
          onChange={(quick_links) => change({ quick_links })}
        />
        <LinksEditor
          title="Social links"
          links={footer.social_links}
          onChange={(social_links) => change({ social_links })}
        />
        <Alert tone="info">
          Every page also shows “{DEVELOPER.label}”. It is part of the portal and is not edited here.
        </Alert>
      </div>
    </Card>
  )
}

function MoreTab({ content, set }) {
  return (
    <div className="flex flex-col gap-4">
      <Card hover={false}>
        <CardHeader title="Sign-in page" subtitle="Leave a field empty to keep the built-in text." />
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <Input
            label="Sign-in title"
            value={content.sign_in.title}
            maxLength={120}
            onChange={(event) =>
              set({ ...content, sign_in: { ...content.sign_in, title: event.target.value } })
            }
          />
          <Input
            label="Sign-in text"
            value={content.sign_in.text}
            maxLength={300}
            onChange={(event) =>
              set({ ...content, sign_in: { ...content.sign_in, text: event.target.value } })
            }
          />
        </div>
      </Card>
      <Card hover={false}>
        <CardHeader title="Search and sharing" subtitle="What search engines and shared links show." />
        <div className="mt-4 flex flex-col gap-4">
          <Input
            label="Page title"
            value={content.seo.title}
            maxLength={120}
            onChange={(event) => set({ ...content, seo: { ...content.seo, title: event.target.value } })}
          />
          <Input
            label="Description"
            value={content.seo.description}
            maxLength={300}
            onChange={(event) =>
              set({ ...content, seo: { ...content.seo, description: event.target.value } })
            }
          />
          <ImageField
            label="Share image"
            value={content.seo.image_url}
            onChange={(image_url) => set({ ...content, seo: { ...content.seo, image_url } })}
          />
        </div>
      </Card>
    </div>
  )
}

function PagesTab({ content, set }) {
  const confirm = useConfirm()
  const [title, setTitle] = useState('')
  const [open, setOpen] = useState('')
  const pages = content.pages
  const slugs = pages.map((page) => page.slug)
  const fresh = slugFrom(title)
  const freshProblem = title.trim() === '' ? '' : slugProblem(fresh, slugs)
  const change = (index, patch) =>
    set({ ...content, pages: pages.map((page, at) => (at === index ? { ...page, ...patch } : page)) })
  const add = () => {
    set({
      ...content,
      pages: [...pages, { slug: fresh, title: title.trim(), seo_title: '', seo_description: '', blocks: [] }],
    })
    setOpen(fresh)
    setTitle('')
  }
  const remove = async (index) => {
    const page = pages[index]
    const ok = await confirm({
      title: `Delete the page “${page.title}”?`,
      message: `${paths.page(page.slug)} stops working once you publish. Links to it will no longer open.`,
      confirmLabel: 'Delete the page',
    })
    if (!ok) return
    set({ ...content, pages: pages.filter((_, at) => at !== index) })
  }
  return (
    <Card hover={false}>
      <CardHeader
        title="Pages"
        subtitle="Each page lives at /p/its-address and is built from the same blocks as a home section."
      />
      <div className="mt-4 flex flex-wrap items-end gap-3">
        <Input
          label="Title of a new page"
          value={title}
          maxLength={300}
          error={freshProblem}
          hint={fresh && !freshProblem ? `It will open at ${paths.page(fresh)}` : ''}
          onChange={(event) => setTitle(event.target.value)}
          className="min-w-56 flex-1"
        />
        <Button variant="secondary" disabled={title.trim() === '' || Boolean(freshProblem)} onClick={add}>
          <Plus size={16} />
          Add page
        </Button>
      </div>
      <div className="mt-4 flex flex-col gap-4">
        {pages.length === 0 && <p className="text-sm text-muted">No pages yet.</p>}
        {pages.map((page, index) => {
          const problem = slugProblem(
            page.slug,
            slugs.filter((_, at) => at !== index),
          )
          return (
            <section key={`page-${index}`} data-page-editor={page.slug} className="rounded-card border border-line p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="wrap-long text-sm font-semibold text-ink">
                  {page.title || 'Untitled page'} · {paths.page(page.slug)}
                </p>
                <span className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setOpen(open === page.slug ? '' : page.slug)}
                  >
                    {open === page.slug ? 'Close' : 'Edit'}
                  </Button>
                  <Button variant="danger" size="sm" onClick={() => remove(index)}>
                    Delete
                  </Button>
                </span>
              </div>
              {open === page.slug && (
                <div className="mt-4 flex flex-col gap-4">
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Input
                      label="Page title"
                      value={page.title}
                      maxLength={300}
                      onChange={(event) => change(index, { title: event.target.value })}
                    />
                    <Input
                      label="Address"
                      value={page.slug}
                      maxLength={60}
                      error={problem}
                      hint={problem ? '' : `Opens at ${paths.page(page.slug)}`}
                      onChange={(event) => {
                        change(index, { slug: event.target.value })
                        setOpen(event.target.value)
                      }}
                    />
                    <Input
                      label="Title for search engines"
                      value={page.seo_title}
                      maxLength={120}
                      onChange={(event) => change(index, { seo_title: event.target.value })}
                    />
                    <Input
                      label="Description for search engines"
                      value={page.seo_description}
                      maxLength={300}
                      onChange={(event) => change(index, { seo_description: event.target.value })}
                    />
                  </div>
                  <BlocksEditor blocks={page.blocks} onChange={(blocks) => change(index, { blocks })} />
                </div>
              )}
            </section>
          )
        })}
      </div>
    </Card>
  )
}

function VersionsTab({ onRestored }) {
  const toast = useToast()
  const confirm = useConfirm()
  const versions = useResource('/api/super-admin/site/versions')
  const [busy, setBusy] = useState(0)
  const restore = async (version) => {
    const ok = await confirm({
      title: `Restore version ${version.id}?`,
      message: 'It becomes the live website at once and replaces the current draft.',
      confirmLabel: 'Restore',
    })
    if (!ok) return
    setBusy(version.id)
    try {
      const answer = await api.post(`/api/super-admin/site/versions/${version.id}/restore`)
      toast.success(answer.message)
      forgetSiteContent()
      versions.reload()
      onRestored()
    } catch (failure) {
      if (!failure?.cancelled) toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const items = versions.data?.items || []
  return (
    <Card hover={false}>
      <CardHeader title="Published versions" subtitle="Every publish is kept with who did it and when." />
      {versions.loading ? (
        <p className="mt-4 text-sm text-muted">Loading the versions…</p>
      ) : items.length === 0 ? (
        <p className="mt-4 text-sm text-muted">
          Nothing is published yet, so visitors see the built-in website.
        </p>
      ) : (
        <ul className="mt-4 flex flex-col divide-y divide-line">
          {items.map((version) => (
            <li
              key={version.id}
              data-version={version.id}
              className="flex flex-wrap items-center justify-between gap-3 py-3"
            >
              <span className="min-w-0">
                <span className="block text-sm font-semibold text-ink">
                  Version {version.id}
                  {version.live ? ' · live now' : ''}
                </span>
                <span className="block wrap-long text-xs text-muted">
                  {formatDateTime(version.saved_at)} · {version.saved_by || 'a removed account'}
                  {version.note ? ` · ${version.note}` : ''}
                </span>
              </span>
              {!version.live && (
                <Button
                  variant="secondary"
                  size="sm"
                  loading={busy === version.id}
                  onClick={() => restore(version)}
                >
                  Restore
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

export default function ControlWebsitePage() {
  const toast = useToast()
  const draft = useResource('/api/super-admin/site/draft')
  const [tab, setTab] = useState('header')
  const [content, setContent] = useState(null)
  const [dirty, setDirty] = useState(false)
  const [busy, setBusy] = useState('')
  const loaded = draft.data
  useEffect(() => {
    if (!loaded) return
    setContent(withDefaults(stripCredit(loaded.content)))
    setDirty(false)
  }, [loaded])
  const set = (next) => {
    setContent(next)
    setDirty(true)
  }
  const save = async () => {
    await api.put('/api/super-admin/site/draft', stripCredit(content))
    setDirty(false)
  }
  const run = async (name, work) => {
    setBusy(name)
    try {
      await work()
    } catch (failure) {
      if (!failure?.cancelled) toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const saveDraft = () =>
    run('save', async () => {
      await save()
      toast.success('The draft is saved. Publish it to make it live.')
    })
  const preview = () =>
    run('preview', async () => {
      const tabWindow = window.open('about:blank', '_blank')
      try {
        if (dirty) await save()
        if (tabWindow) tabWindow.location.href = `${paths.root()}?preview=1`
      } catch (failure) {
        if (tabWindow) tabWindow.close()
        throw failure
      }
    })
  const publish = () =>
    run('publish', async () => {
      await save()
      const answer = await api.post('/api/super-admin/site/publish')
      forgetSiteContent()
      toast.success(answer.message)
      draft.reload()
    })
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Website"
        subtitle="Edit the public website, preview the draft and publish it"
        tabs={<StatusTabs tabs={TABS} value={tab} onChange={setTab} tone="header" />}
      />
      {draft.error && (
        <Alert tone="error" className="mb-5">
          {draft.error}
        </Alert>
      )}
      {!content ? (
        <SkeletonCard />
      ) : (
        <div className="flex flex-col gap-4">
          <Card hover={false} data-website-actions>
            <div className="flex flex-wrap items-center gap-3">
              <p className="min-w-0 flex-1 text-sm text-muted" data-website-state>
                {dirty
                  ? 'You have changes that are not saved yet.'
                  : loaded?.draft
                    ? 'This is the saved draft. Visitors see it after you publish.'
                    : loaded?.fallback
                      ? 'Nothing is published yet. Visitors see the built-in website.'
                      : 'This is the published website.'}
              </p>
              <Button variant="secondary" loading={busy === 'save'} disabled={!dirty} onClick={saveDraft}>
                <Save size={16} />
                Save draft
              </Button>
              <Button variant="secondary" loading={busy === 'preview'} onClick={preview}>
                <Eye size={16} />
                Preview
              </Button>
              <Button loading={busy === 'publish'} onClick={publish}>
                <UploadCloud size={16} />
                Publish
              </Button>
            </div>
          </Card>
          {tab === 'header' && <HeaderTab content={content} set={set} />}
          {tab === 'home' && <HomeTab content={content} set={set} />}
          {tab === 'sections' && <SectionsTab content={content} set={set} />}
          {tab === 'footer' && <FooterTab content={content} set={set} />}
          {tab === 'more' && <MoreTab content={content} set={set} />}
          {tab === 'pages' && <PagesTab content={content} set={set} />}
          {tab === 'versions' && <VersionsTab onRestored={draft.reload} />}
        </div>
      )}
    </>
  )
}
