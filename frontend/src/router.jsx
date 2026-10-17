import { Navigate, Route, Routes } from 'react-router-dom'
import AppLayout from './components/layout/AppLayout'
import AdminLayout from './components/layout/AdminLayout'
import RequireRole from './components/RequireRole'
import LoginPage from './pages/auth/LoginPage'
import AdminLoginPage from './pages/auth/AdminLoginPage'
import RegisterPage from './pages/auth/RegisterPage'
import ForgotPasswordPage from './pages/auth/ForgotPasswordPage'
import UpcomingRoutinePage from './pages/student/UpcomingRoutinePage'
import DropCourseRoutinePage from './pages/student/DropCourseRoutinePage'
import ExamsPage from './pages/student/ExamsPage'
import FixedRoutinePage from './pages/student/FixedRoutinePage'
import CRNoticesPage from './pages/student/CRNoticesPage'
import AdminNoticesPage from './pages/student/AdminNoticesPage'
import MaterialsPage from './pages/student/MaterialsPage'
import StudentsDirectoryPage from './pages/student/StudentsDirectoryPage'
import PollsPage from './pages/student/PollsPage'
import TeachersPage from './pages/student/TeachersPage'
import MessagesPage from './pages/student/MessagesPage'
import NotFoundPage from './pages/NotFoundPage'
import Placeholder from './pages/Placeholder'
import { useAuth } from './context/auth'
import { homeFor, paths } from './lib/paths'

const page = (title, subtitle) => <Placeholder title={title} subtitle={subtitle} />

function RootRedirect() {
  const { isAuthenticated, role } = useAuth()
  return <Navigate to={isAuthenticated ? homeFor(role) : paths.login()} replace />
}

export default function Router() {
  return (
    <Routes>
      <Route path={paths.root()} element={<RootRedirect />} />
      <Route path={paths.login()} element={<LoginPage />} />
      <Route path={paths.register()} element={<RegisterPage />} />
      <Route path={paths.forgotPassword()} element={<ForgotPasswordPage />} />
      <Route path={paths.adminLogin()} element={<AdminLoginPage />} />
      <Route path={paths.adminForgotPassword()} element={<ForgotPasswordPage admin />} />
      <Route element={<RequireRole roles="student" />}>
        <Route element={<AppLayout />}>
          <Route
            path={paths.student.home()}
            element={<Navigate to={paths.student.upcomingRoutine()} replace />}
          />
          <Route path={paths.student.upcomingRoutine()} element={<UpcomingRoutinePage />} />
          <Route path={paths.student.dropCourseRoutine()} element={<DropCourseRoutinePage />} />
          <Route path={paths.student.exams()} element={<ExamsPage />} />
          <Route path={paths.student.fixedRoutine()} element={<FixedRoutinePage />} />
          <Route path={paths.student.crNotices()} element={<CRNoticesPage />} />
          <Route path={paths.student.adminNotices()} element={<AdminNoticesPage />} />
          <Route path={paths.student.materials()} element={<MaterialsPage />} />
          <Route path={paths.student.students()} element={<StudentsDirectoryPage />} />
          <Route path={paths.student.polls()} element={<PollsPage />} />
          <Route path={paths.student.teachers()} element={<TeachersPage />} />
          <Route path={paths.student.messages()} element={<MessagesPage />} />
        </Route>
      </Route>
      <Route element={<RequireRole roles="cr" />}>
        <Route element={<AppLayout />}>
          <Route
            path={paths.cr.home()}
            element={<Navigate to={paths.cr.fixedRoutine()} replace />}
          />
          <Route
            path={paths.cr.fixedRoutine()}
            element={page('Fixed Routine', 'The weekly routine of your batch and section')}
          />
          <Route
            path={paths.cr.createClass()}
            element={page('Create Class/Exam', 'Ask a teacher to approve a class or an exam')}
          />
          <Route
            path={paths.cr.extraClass()}
            element={page('Extra Class', 'Request an extra class outside the routine')}
          />
          <Route
            path={paths.cr.requests()}
            element={page('My Requests', 'What you asked for and where it stands')}
          />
          <Route
            path={paths.cr.sendMessage()}
            element={page('Send Message', 'Write to your batch, another batch or a teacher')}
          />
          <Route
            path={paths.cr.teacherMessages()}
            element={page('Teacher Messages', 'Messages teachers sent you')}
          />
          <Route path={paths.cr.notices()} element={page('Notices', 'Notices you posted')} />
          <Route path={paths.cr.materials()} element={page('Materials', 'Links you shared')} />
          <Route path={paths.cr.polls()} element={page('Polls', 'Polls you created')} />
          <Route
            path={paths.cr.teachers()}
            element={page('Teachers', 'Contact details of the department teachers')}
          />
        </Route>
      </Route>
      <Route element={<RequireRole roles="teacher" />}>
        <Route element={<AppLayout />}>
          <Route
            path={paths.teacher.home()}
            element={<Navigate to={paths.teacher.schedule()} replace />}
          />
          <Route
            path={paths.teacher.schedule()}
            element={page('Daily Schedule', 'Your classes for the chosen day')}
          />
          <Route
            path={paths.teacher.requests()}
            element={page('Approval Requests', 'Requests waiting for your decision')}
          />
          <Route
            path={paths.teacher.cancelClass()}
            element={page('Cancel Class', 'Cancel one of your approved classes')}
          />
          <Route
            path={paths.teacher.crMessages()}
            element={page('CR Messages', 'Messages class representatives sent you')}
          />
          <Route
            path={paths.teacher.sendMessage()}
            element={page('Send Message', 'Write to a class representative, a batch or a student')}
          />
        </Route>
      </Route>
      <Route element={<RequireRole roles="admin" />}>
        <Route element={<AdminLayout />}>
          <Route
            path={paths.admin.overview()}
            element={page('Overview', 'How the department is doing right now')}
          />
          <Route
            path={paths.admin.registrations()}
            element={page('Registrations', 'Student accounts waiting for approval')}
          />
          <Route path={paths.admin.students()} element={page('Students', 'Every student account')} />
          <Route path={paths.admin.crs()} element={page('CRs', 'Class representative accounts')} />
          <Route path={paths.admin.teachers()} element={page('Teachers', 'Teacher accounts')} />
          <Route path={paths.admin.courses()} element={page('Courses', 'The course catalogue')} />
          <Route
            path={paths.admin.routines()}
            element={page('Routines', 'The fixed weekly routine')}
          />
          <Route
            path={paths.admin.notices()}
            element={page('Notices', 'Notices from the administration')}
          />
          <Route
            path={paths.admin.materials()}
            element={page('Materials', 'Links shared with the students')}
          />
        </Route>
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
