import { Link } from 'react-router-dom'
import SiteFooter from '../../components/home/SiteFooter'
import PublicHeader from '../../components/site/PublicHeader'
import { isNative } from '../../lib/native'
import paths from '../../lib/paths'
import { DEVELOPER } from '../../lib/siteContent'
import { useSeo, useSiteContent } from '../../lib/useSiteContent'

export default function AboutDevelopersPage() {
  const { content, preview } = useSiteContent()
  useSeo({ title: `About ${DEVELOPER.name}`, description: `The team that built this portal.` })
  const app = isNative()
  return (
    <div className="flex min-h-dvh flex-col bg-paper">
      {!app && <PublicHeader content={content} preview={preview} />}
      <main className="flex-1 px-4 py-10 sm:px-6 sm:py-14">
        <article data-about-developers className="mx-auto max-w-2xl">
          <p className="eyebrow">The developers</p>
          <h1 className="font-display mt-2 text-3xl font-bold text-ink">About {DEVELOPER.name}</h1>
          <p className="mt-4 text-sm leading-relaxed text-muted">
            {DEVELOPER.name} designed and built this class and exam coordination portal, its website
            and its Android app for the Department of Computer Science and Engineering.
          </p>
          <h2 className="font-display mt-8 text-lg font-semibold text-ink">The team</h2>
          <ul className="mt-3 flex flex-col gap-2">
            {DEVELOPER.people.map((person) => (
              <li
                key={person}
                className="rounded-card border border-line bg-surface px-4 py-3 text-sm font-semibold text-ink"
              >
                {person}
              </li>
            ))}
          </ul>
          <Link
            to={paths.root()}
            className="mt-8 inline-block text-sm font-semibold text-brand-600 underline"
          >
            {app ? 'Back to the app' : 'Back to the home page'}
          </Link>
        </article>
      </main>
      {!app && <SiteFooter content={content} />}
    </div>
  )
}
