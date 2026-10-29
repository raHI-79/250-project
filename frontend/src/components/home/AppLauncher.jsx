import { Link } from 'react-router-dom'
import { GraduationCap, Users } from 'lucide-react'
import { Button, LogoBadge } from '../ui'
import AndroidDownload from './AndroidDownload'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import paths from '../../lib/paths'

export default function AppLauncher() {
  return (
    <div className="flex min-h-dvh items-center justify-center px-5 py-[max(2rem,env(safe-area-inset-top,0px))]">
      <div className="glass-sheet animate-fade-in-up w-full max-w-sm rounded-sheet px-6 py-9 text-center">
        <LogoBadge size="xl" className="mx-auto" />
        <h1 className="font-display mt-6 text-title font-bold text-ink">SUST CSE</h1>
        <p className="mt-2 text-sm text-muted">{DEPARTMENT}</p>
        <p className="mt-1 text-2xs tracking-[0.1em] text-faint uppercase">{UNIVERSITY}</p>
        <div className="mt-8 flex flex-col gap-3">
          <Button as={Link} to={paths.studentLogin()} size="lg" className="w-full">
            <Users size={18} />
            Student sign in
          </Button>
          <Button
            as={Link}
            to={paths.teacherLogin()}
            variant="secondary"
            size="lg"
            className="w-full"
          >
            <GraduationCap size={18} />
            Teacher sign in
          </Button>
          <AndroidDownload variant="ghost" className="w-full" />
        </div>
        <p className="mt-6 text-xs text-muted">
          New student?{' '}
          <Link to={paths.studentRegister()} className="font-semibold text-brand-600 underline">
            Register with your university email
          </Link>
        </p>
      </div>
    </div>
  )
}
