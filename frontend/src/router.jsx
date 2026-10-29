import { Navigate, Route, Routes } from 'react-router-dom'
import AppLayout from './components/layout/AppLayout'
import AdminLayout from './components/layout/AdminLayout'
import RequireRole, { RequireAuth } from './components/RequireRole'
import LoginPage from './pages/auth/LoginPage'
import AdminLoginPage from './pages/auth/AdminLoginPage'
import RegisterPage from './pages/auth/RegisterPage'
import ForgotPasswordPage from './pages/auth/ForgotPasswordPage'
import ActivatePage from './pages/auth/ActivatePage'
import HomePage from './pages/HomePage'
import ProfilePage from './pages/ProfilePage'
import PersonPage from './pages/PersonPage'
import TodayPage from './pages/student/TodayPage'
import UpcomingRoutinePage from './pages/student/UpcomingRoutinePage'
import ExamsPage from './pages/student/ExamsPage'
import CoursesStudentPage from './pages/student/CoursesPage'
import FixedRoutinePage from './pages/student/FixedRoutinePage'
import NoticesPage from './pages/NoticesPage'
import MaterialsPage from './pages/student/MaterialsPage'
import StudentsDirectoryPage from './pages/student/StudentsDirectoryPage'
import PollsPage from './pages/student/PollsPage'
import StudentAttendancePage from './pages/student/StudentAttendancePage'
import MyAttendancePage from './pages/student/MyAttendancePage'
import TeachersPage from './pages/student/TeachersPage'
import MessagesPage from './pages/MessagesPage'
import CreateClassExamPage from './pages/cr/CreateClassExamPage'
import CRPostNoticesPage from './pages/cr/CRNoticesPage'
import CRDriveLinkPage from './pages/cr/CRDriveLinkPage'
import CRPollsPage from './pages/cr/CRPollsPage'
import TeacherFixedRoutinePage from './pages/teacher/TeacherFixedRoutinePage'
import TeacherUpcomingPage from './pages/teacher/TeacherUpcomingPage'
import TeacherCreateExamPage from './pages/teacher/TeacherCreateExamPage'
import TeacherNoticesPage from './pages/teacher/TeacherNoticesPage'
import TeacherAttendancePage from './pages/teacher/TeacherAttendancePage'
import OverviewPage from './pages/admin/OverviewPage'
import RegistrationsPage from './pages/admin/RegistrationsPage'
import AdminStudentsPage from './pages/admin/AdminStudentsPage'
import CRsPage from './pages/admin/CRsPage'
import AdminTeachersPage from './pages/admin/AdminTeachersPage'
import CoursesPage from './pages/admin/CoursesPage'
import ClassPeriodsPage from './pages/admin/ClassPeriodsPage'
import AdminRoutinesPage from './pages/admin/AdminRoutinesPage'
import AdminPostNoticesPage from './pages/admin/AdminPostNoticesPage'
import ActivityLogPage from './pages/admin/ActivityLogPage'
import NotFoundPage from './pages/NotFoundPage'
import { useAuth } from './context/auth'
import { isNative } from './lib/native'
import { useNativeBackButton } from './lib/shell'
import { PORTAL_ADMIN, PORTAL_STUDENT, PORTAL_TEACHER, homeFor, paths } from './lib/paths'

const STUDENT_ROLES = ['student', 'cr']

function Root() {
  const { isAuthenticated, role } = useAuth()
  if (isAuthenticated) return <Navigate to={homeFor(role)} replace />
  if (isNative()) return <Navigate to={paths.studentLogin()} replace />
  return <HomePage />
}

export default function Router() {
  useNativeBackButton()
  const office = !isNative()
  return (
    <Routes>
      <Route path={paths.root()} element={<Root />} />
      <Route path={paths.studentLogin()} element={<LoginPage portal={PORTAL_STUDENT} />} />
      <Route path={paths.studentRegister()} element={<RegisterPage />} />
      <Route
        path={paths.studentForgotPassword()}
        element={<ForgotPasswordPage portal={PORTAL_STUDENT} />}
      />
      <Route path={paths.teacherLogin()} element={<LoginPage portal={PORTAL_TEACHER} />} />
      <Route
        path={paths.teacherForgotPassword()}
        element={<ForgotPasswordPage portal={PORTAL_TEACHER} />}
      />
      <Route path={paths.teacherActivate()} element={<ActivatePage />} />
      {office ? (
        <>
          <Route path={paths.adminLogin()} element={<AdminLoginPage />} />
          <Route
            path={paths.adminForgotPassword()}
            element={<ForgotPasswordPage portal={PORTAL_ADMIN} />}
          />
        </>
      ) : (
        <Route path={`${paths.adminLogin()}/*`} element={<Navigate to={paths.root()} replace />} />
      )}
      <Route element={<RequireAuth />}>
        <Route element={<AppLayout />}>
          <Route path={paths.profile()} element={<ProfilePage />} />
          <Route path={paths.personPattern()} element={<PersonPage />} />
        </Route>
      </Route>
      <Route element={<RequireRole roles={STUDENT_ROLES} />}>
        <Route element={<AppLayout />}>
          <Route
            path={paths.student.home()}
            element={<Navigate to={paths.student.today()} replace />}
          />
          <Route path={paths.student.today()} element={<TodayPage />} />
          <Route path={paths.student.upcomingRoutine()} element={<UpcomingRoutinePage />} />
          <Route path={paths.student.exams()} element={<ExamsPage />} />
          <Route path={paths.student.fixedRoutine()} element={<FixedRoutinePage />} />
          <Route path={paths.student.courses()} element={<CoursesStudentPage />} />
          <Route path={paths.student.notices()} element={<NoticesPage />} />
          <Route path={paths.student.materials()} element={<MaterialsPage />} />
          <Route path={paths.student.students()} element={<StudentsDirectoryPage />} />
          <Route path={paths.student.polls()} element={<PollsPage />} />
          <Route path={paths.student.attendance()} element={<StudentAttendancePage />} />
          <Route path={paths.student.myAttendance()} element={<MyAttendancePage />} />
          <Route path={paths.student.teachers()} element={<TeachersPage />} />
          <Route path={paths.student.messages()} element={<MessagesPage />} />
        </Route>
      </Route>
      <Route element={<RequireRole roles="cr" />}>
        <Route element={<AppLayout />}>
          <Route
            path={paths.cr.home()}
            element={<Navigate to={paths.student.today()} replace />}
          />
          <Route path={paths.cr.schedule()} element={<CreateClassExamPage />} />
          <Route path={paths.cr.postNotices()} element={<CRPostNoticesPage />} />
          <Route path={paths.cr.driveLink()} element={<CRDriveLinkPage />} />
          <Route path={paths.cr.polls()} element={<CRPollsPage />} />
        </Route>
      </Route>
      <Route element={<RequireRole roles="teacher" />}>
        <Route element={<AppLayout />}>
          <Route
            path={paths.teacher.home()}
            element={<Navigate to={paths.teacher.upcoming()} replace />}
          />
          <Route path={paths.teacher.fixedRoutine()} element={<TeacherFixedRoutinePage />} />
          <Route path={paths.teacher.upcoming()} element={<TeacherUpcomingPage />} />
          <Route path={paths.teacher.createExam()} element={<TeacherCreateExamPage />} />
          <Route path={paths.teacher.notices()} element={<TeacherNoticesPage />} />
          <Route path={paths.teacher.attendance()} element={<TeacherAttendancePage />} />
          <Route path={paths.teacher.messages()} element={<MessagesPage />} />
        </Route>
      </Route>
      {office && (
      <Route element={<RequireRole roles="admin" />}>
        <Route element={<AdminLayout />}>
          <Route path={paths.admin.overview()} element={<OverviewPage />} />
          <Route path={paths.admin.registrations()} element={<RegistrationsPage />} />
          <Route path={paths.admin.students()} element={<AdminStudentsPage />} />
          <Route path={paths.admin.crs()} element={<CRsPage />} />
          <Route path={paths.admin.teachers()} element={<AdminTeachersPage />} />
          <Route path={paths.admin.courses()} element={<CoursesPage />} />
          <Route path={paths.admin.periods()} element={<ClassPeriodsPage />} />
          <Route path={paths.admin.routines()} element={<AdminRoutinesPage />} />
          <Route path={paths.admin.notices()} element={<AdminPostNoticesPage />} />
          <Route path={paths.admin.messages()} element={<MessagesPage />} />
          <Route path={paths.admin.activity()} element={<ActivityLogPage />} />
        </Route>
      </Route>
      )}
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
