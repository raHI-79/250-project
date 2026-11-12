import { Link, useParams } from 'react-router-dom'
import SiteFooter from '../../components/home/SiteFooter'
import Blocks from '../../components/site/Blocks'
import PublicHeader from '../../components/site/PublicHeader'
import paths from '../../lib/paths'
import { pageBySlug, seoFor } from '../../lib/siteContent'
import { useSeo, useSiteContent } from '../../lib/useSiteContent'

export default function CustomPage() {
  const { slug } = useParams()
  const { content, preview, loading } = useSiteContent()
  const page = pageBySlug(content, slug)
  useSeo(page ? seoFor(content, page) : null)
  return (
    <div className="flex min-h-dvh flex-col bg-paper">
      <PublicHeader content={content} preview={preview} />
      <main className="flex-1 px-4 py-10 sm:px-6 sm:py-14">
        <div className="mx-auto max-w-4xl">
          {page ? (
            <article data-site-page={page.slug}>
              <h1 className="font-display wrap-long text-3xl font-bold text-ink">{page.title}</h1>
              <Blocks blocks={page.blocks} className="mt-8" />
            </article>
          ) : loading ? (
            <p className="text-sm text-muted">Loading the page…</p>
          ) : (
            <div data-site-page-missing>
              <h1 className="font-display text-2xl font-bold text-ink">This page does not exist</h1>
              <p className="mt-2 text-sm text-muted">
                It may have been removed, or the address is not right.
              </p>
              <Link
                to={paths.root()}
                className="mt-5 inline-block text-sm font-semibold text-brand-600 underline"
              >
                Go to the home page
              </Link>
            </div>
          )}
        </div>
      </main>
      <SiteFooter content={content} />
    </div>
  )
}
