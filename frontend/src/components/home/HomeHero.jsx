import { Link } from 'react-router-dom'
import { HOME_PHOTO } from '../../lib/campus'
import paths from '../../lib/paths'
import { Button } from '../ui'
import AndroidDownload from './AndroidDownload'

export default function HomeHero() {
  return (
    <section className="relative isolate overflow-hidden bg-deep-950">
      <img
        src={HOME_PHOTO.src}
        alt=""
        aria-hidden="true"
        decoding="async"
        fetchPriority="high"
        sizes="100vw"
        className="absolute inset-0 size-full object-cover"
      />
      <div className="absolute inset-0 bg-linear-to-r from-deep-950/85 via-deep-950/55 to-deep-900/25" />
      <div className="relative mx-auto flex min-h-[26rem] max-w-6xl flex-col justify-center px-4 py-14 sm:px-6 sm:py-20 lg:min-h-[32.5rem]">
        <p className="text-xs font-semibold tracking-[0.14em] text-sand uppercase">
          Established 1991 · Sylhet, Bangladesh
        </p>
        <h1 className="font-display mt-4 max-w-2xl text-3xl leading-tight font-semibold text-white sm:text-4xl lg:text-5xl">
          Class and exam coordination for the <em className="text-sand not-italic">CSE</em>{' '}
          department
        </h1>
        <p className="mt-4 max-w-xl text-sm leading-relaxed text-white/80 sm:text-base">
          One record for the weekly routine, rescheduled classes, exams and notices. Class
          representatives draft the plan, teachers confirm it, and every student reads the published
          result.
        </p>
        <div className="mt-7 flex flex-wrap gap-3">
          <Button as={Link} to={paths.studentLogin()} size="lg" variant="light">
            Student sign in
          </Button>
          <Button as={Link} to={paths.teacherLogin()} size="lg" variant="outlineLight">
            Teacher sign in
          </Button>
          <AndroidDownload variant="outlineLight" />
        </div>
        <p className="mt-10 text-xs text-white/55">{HOME_PHOTO.caption}</p>
      </div>
    </section>
  )
}
