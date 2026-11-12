import { Link } from 'react-router-dom'
import {
  BellRing,
  CalendarClock,
  ClipboardList,
  FolderOpen,
  MessagesSquare,
  Vote,
} from 'lucide-react'
import AppLauncher from '../components/home/AppLauncher'
import HomeHero from '../components/home/HomeHero'
import SiteFooter from '../components/home/SiteFooter'
import StatsStrip from '../components/home/StatsStrip'
import AndroidDownload from '../components/home/AndroidDownload'
import Blocks from '../components/site/Blocks'
import PublicHeader from '../components/site/PublicHeader'
import RichText from '../components/site/RichText'
import { seoFor, visibleSections, withDefaults } from '../lib/siteContent'
import { useSeo, useSiteContent } from '../lib/useSiteContent'
import paths from '../lib/paths'
import { useStandalone } from '../lib/standalone'

const FEATURES = [
  {
    icon: ClipboardList,
    title: 'Fixed weekly routine',
    text: 'The routine the office enters is the single source for every batch, section and room.',
  },
  {
    icon: CalendarClock,
    title: 'Classes and exams ahead',
    text: 'Class representatives draft extra classes, reschedules and exams for the days ahead.',
  },
  {
    icon: BellRing,
    title: 'Teacher confirmation',
    text: 'Each teacher confirms, cancels or asks to move a class before it reaches the students.',
  },
  {
    icon: FolderOpen,
    title: 'Notices and materials',
    text: 'Office, teacher and CR notices sit together with the drive links for each course.',
  },
  {
    icon: Vote,
    title: 'Batch polls',
    text: 'CRs ask their batch about exam dates or class timings and read the result at once.',
  },
  {
    icon: MessagesSquare,
    title: 'Direct messages',
    text: 'Students, CRs and teachers reach each other without leaving the department record.',
  },
]

const STEPS = [
  {
    title: 'Register with your university email',
    text: 'Students sign up with their registration number and verify their student email address.',
  },
  {
    title: 'The office approves the account',
    text: 'Teachers receive an invitation and choose their own user ID while activating it.',
  },
  {
    title: 'The plan is drafted and confirmed',
    text: 'A CR schedules the class or exam, the teacher confirms it, and the room is held.',
  },
  {
    title: 'The routine is published',
    text: 'Students read the published day, the upcoming routine and every exam in one place.',
  },
]

const PORTALS = [
  {
    letter: 'S',
    title: 'Student portal',
    text: 'Today, the published routine, exams, notices, materials and polls for your batch.',
    to: paths.studentLogin(),
  },
  {
    letter: 'T',
    title: 'Teacher portal',
    text: 'Your weekly routine, the classes awaiting your decision and the exams you set.',
    to: paths.teacherLogin(),
  },
]

function AboutSection({ content }) {
  const home = withDefaults(content).home
  return (
    <section data-home-section="about" className="px-4 py-14 sm:px-6 sm:py-16">
      <div className="mx-auto max-w-6xl">
        <p className="eyebrow">What the department runs here</p>
        {home.about_title && (
          <h2 className="font-display mt-2 wrap-long text-[22px] font-bold text-ink sm:text-2xl">
            {home.about_title}
          </h2>
        )}
        <RichText text={home.about_text} className="mt-3 max-w-3xl text-muted" />
        <div className="mt-8 grid gap-px bg-line sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((feature, position) => (
            <article key={feature.title} className="border-t-[3px] border-brand-600 bg-surface p-5">
              <div className="flex items-center justify-between gap-3">
                <feature.icon size={18} className="text-brand-600" />
                <span className="font-numeral text-lg text-gold-500">
                  {String(position + 1).padStart(2, '0')}
                </span>
              </div>
              <h3 className="font-display mt-3 text-base font-semibold text-ink">{feature.title}</h3>
              <p className="mt-1.5 text-sm leading-relaxed text-muted">{feature.text}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}

function HomeSection({ entry, content }) {
  if (entry.key === 'stats') return <StatsStrip />
  if (entry.key === 'about') return <AboutSection content={content} />
  if (entry.key === 'notices') {
    return (
      <section data-home-section="notices" className="border-y border-line bg-wash px-4 py-10 sm:px-6">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="eyebrow">Notice board</p>
            <h2 className="font-display mt-2 text-xl font-bold text-ink">
              Department, teacher and CR notices
            </h2>
            <p className="mt-1 text-sm text-muted">
              Notices are for the students and teachers of the department. Sign in to read them.
            </p>
          </div>
          <Link
            to={paths.studentLogin()}
            className="press inline-flex min-h-11 items-center rounded-xl border border-brand-600 px-4 text-sm font-semibold text-brand-600 hover:bg-brand-50"
          >
            Sign in to read the notices
          </Link>
        </div>
      </section>
    )
  }
  if (entry.key === 'app') {
    return (
      <section data-home-section="app" className="px-4 py-10 sm:px-6">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 rounded-card border border-line bg-surface p-5">
          <div className="min-w-0">
            <h2 className="font-display text-xl font-bold text-ink">The portal as an Android app</h2>
            <p className="mt-1 text-sm text-muted">
              Attendance by Bluetooth, notifications and the routine of the day on your phone.
            </p>
          </div>
          <AndroidDownload variant="primary" size="md" />
        </div>
      </section>
    )
  }
  if (!entry.custom) return null
  return (
    <section data-home-section={entry.key} className="px-4 py-14 sm:px-6 sm:py-16">
      <div className="mx-auto max-w-6xl">
        <h2 className="font-display wrap-long text-[22px] font-bold text-ink sm:text-2xl">
          {entry.custom.title}
        </h2>
        <Blocks blocks={entry.custom.blocks} className="mt-6" />
      </div>
    </section>
  )
}

export default function HomePage() {
  const installed = useStandalone()
  const { content, preview } = useSiteContent()
  useSeo(seoFor(content))
  if (installed) return <AppLauncher />
  const sections = visibleSections(content)
  return (
    <div className="min-h-dvh bg-paper">
      <PublicHeader content={content} preview={preview} />
      <HomeHero content={content} />
      {sections.map((entry) => (
        <HomeSection key={entry.key} entry={entry} content={content} />
      ))}
      <section className="border-y border-line bg-wash px-4 py-14 sm:px-6 sm:py-16">
        <div className="mx-auto max-w-6xl">
          <p className="eyebrow">How it works</p>
          <h2 className="font-display mt-2 text-[22px] font-bold text-ink sm:text-2xl">
            From registration to a published routine
          </h2>
          <ol className="mt-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {STEPS.map((step, position) => (
              <li key={step.title}>
                <span className="font-numeral grid size-11 place-items-center rounded-full border border-brand-600 text-lg text-brand-600">
                  {position + 1}
                </span>
                <h3 className="font-display mt-3 text-base font-semibold text-ink">
                  {step.title}
                </h3>
                <p className="mt-1.5 text-sm leading-relaxed text-muted">{step.text}</p>
              </li>
            ))}
          </ol>
        </div>
      </section>
      <section className="px-4 py-14 sm:px-6 sm:py-16">
        <div className="mx-auto max-w-6xl">
          <p className="eyebrow">Sign in</p>
          <h2 className="font-display mt-2 text-[22px] font-bold text-ink sm:text-2xl">
            Choose your portal
          </h2>
          <div className="mt-8 grid gap-5 lg:grid-cols-2">
            {PORTALS.map((portal) => (
              <Link
                key={portal.title}
                to={portal.to}
                className="group relative isolate overflow-hidden rounded-card border border-line bg-linear-to-br from-deep-900 via-deep-800 to-deep-700 p-6 sm:p-8"
              >
                <span
                  aria-hidden="true"
                  className="font-display pointer-events-none absolute -right-4 -bottom-16 text-[11rem] leading-none text-white/10"
                >
                  {portal.letter}
                </span>
                <h3 className="font-display relative text-xl font-semibold text-white">
                  {portal.title}
                </h3>
                <p className="relative mt-2 max-w-sm text-sm leading-relaxed text-white/75">
                  {portal.text}
                </p>
                <span className="relative mt-6 inline-flex items-center gap-2 text-sm font-semibold text-sand group-hover:text-white">
                  Continue
                </span>
              </Link>
            ))}
          </div>
          <p className="mt-5 text-xs text-muted">
            New student? <Link
              to={paths.studentRegister()}
              className="font-semibold text-brand-600 underline"
            >
              Register with your university email
            </Link>
            . Teachers activate the invitation sent by the office.
          </p>
        </div>
      </section>
      <SiteFooter content={content} />
    </div>
  )
}
