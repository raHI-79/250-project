import { Link } from 'react-router-dom'
import { GraduationCap, Users } from 'lucide-react'
import { BrandMark, Button } from '../ui'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import { SUST_LOGO } from '../../lib/brand'
import paths from '../../lib/paths'

export default function AppLauncher() {
  return (
    <div className="flex min-h-dvh flex-col justify-between bg-page px-6 pt-[max(3rem,env(safe-area-inset-top,0px))] pb-[max(2rem,env(safe-area-inset-bottom,0px))]">
      <div className="flex flex-1 flex-col items-center justify-center text-center">
        <BrandMark src={SUST_LOGO} alt={UNIVERSITY} className="size-24" />
        <h1 className="font-display mt-5 text-title font-bold text-ink">SUST CSE</h1>
        <p className="mt-2 max-w-xs text-sm text-muted">{DEPARTMENT}</p>
        <p className="mt-1 text-xs text-faint">Class and exam coordination</p>
      </div>
      <div className="flex flex-col gap-3">
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
        <p className="mt-2 text-center text-xs text-muted">
          New student?{' '}
          <Link to={paths.studentRegister()} className="font-semibold text-brand-600 underline">
            Register with your university email
          </Link>
        </p>
      </div>
    </div>
  )
}
