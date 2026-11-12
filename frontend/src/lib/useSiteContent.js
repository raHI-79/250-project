import { useEffect, useState } from 'react'
import api, { hasSession } from './api'
import { withDefaults } from './siteContent'

const PREVIEW_PARAM = 'preview'

let published = null

let waiting = null

export const isPreview = (search) => {
  try {
    return new URLSearchParams(search || '').get(PREVIEW_PARAM) === '1'
  } catch {
    return false
  }
}

export const forgetSiteContent = () => {
  published = null
  waiting = null
}

const loadPublished = () => {
  if (published) return Promise.resolve(published)
  if (!waiting) {
    waiting = api
      .get('/api/site/content', { auth: false })
      .then((data) => {
        published = { content: withDefaults(data?.content), fallback: Boolean(data?.fallback) }
        return published
      })
      .catch(() => ({ content: withDefaults(null), fallback: true }))
      .finally(() => {
        waiting = null
      })
  }
  return waiting
}

const loadDraft = () =>
  api.get('/api/super-admin/site/draft').then((data) => ({
    content: withDefaults(data?.content),
    fallback: Boolean(data?.fallback),
    draft: Boolean(data?.draft),
  }))

export const useSiteContent = () => {
  const preview =
    typeof window !== 'undefined' && isPreview(window.location.search) && hasSession()
  const [state, setState] = useState(() => ({
    content: published?.content || withDefaults(null),
    loading: !published || preview,
    preview: false,
    refused: false,
  }))
  useEffect(() => {
    let live = true
    const settle = (next) => {
      if (live) setState(next)
    }
    if (preview) {
      loadDraft()
        .then((data) => settle({ content: data.content, loading: false, preview: true, refused: false }))
        .catch(() =>
          loadPublished().then((data) =>
            settle({ content: data.content, loading: false, preview: false, refused: true }),
          ),
        )
    } else {
      loadPublished().then((data) =>
        settle({ content: data.content, loading: false, preview: false, refused: false }),
      )
    }
    return () => {
      live = false
    }
  }, [preview])
  return state
}

const setMeta = (selector, attribute, name, value) => {
  let node = document.head.querySelector(selector)
  if (!value) {
    if (node && node.dataset.site === '1') node.remove()
    return
  }
  if (!node) {
    node = document.createElement('meta')
    node.setAttribute(attribute, name)
    node.dataset.site = '1'
    document.head.appendChild(node)
  }
  node.setAttribute('content', value)
}

export const useSeo = (seo) => {
  const title = seo?.title || ''
  const description = seo?.description || ''
  const image = seo?.image || ''
  useEffect(() => {
    if (!title) return undefined
    const before = document.title
    document.title = title
    setMeta('meta[name="description"]', 'name', 'description', description)
    setMeta('meta[property="og:title"]', 'property', 'og:title', title)
    setMeta('meta[property="og:description"]', 'property', 'og:description', description)
    setMeta('meta[property="og:image"]', 'property', 'og:image', image)
    return () => {
      document.title = before
    }
  }, [title, description, image])
}
