# Development Roadmap

The project is built in 27 small development sets. Each set is owned by one team
member, is worked on alone, and ends with reviewed Git commits.

- **Owner A** — Ratul Islam ([@raHI-79](https://github.com/raHI-79))
- **Owner B** — TBD

Status values: `todo` → `in progress` → `done`.

Full requirements: [SPEC.md](SPEC.md)

## Sets

| Set | Owner | Title | Status |
|---:|:---:|---|---|
| 1 | A | Monorepo, Go skeleton, config, JSON helpers, health endpoint, guide files | done |
| 2 | A | PostgreSQL connection and migrations for all tables | done |
| 3 | A | Security core: bcrypt, JWT, middleware, admin account seed | done |
| 4 | B | Auth APIs: registration, unified login, admin login, logout, password reset | todo |
| 5 | A | Admin people: approvals, students, CRs, teachers, overview stats | todo |
| 6 | B | Courses, fixed routines with conflict + hours rules, lookup endpoints | todo |
| 7 | B | Notification service and endpoints | todo |
| 8 | A | CR class/exam/extra-class requests, room availability, conflicts, My Requests | todo |
| 9 | B | Teacher approvals, cancel, daily schedule, auto-cancel background job | todo |
| 10 | A | Student routine APIs: upcoming routine, drop course routine, exams, fixed routine filters | todo |
| 11 | B | File uploads + notices (admin and CR) | todo |
| 12 | A | Materials, student directory, teachers info | todo |
| 13 | B | Messages (CR-teacher two-way, broadcasts, student read-only inbox) | todo |
| 14 | A | Polls | todo |
| 15 | B | Backend security/RBAC test suite | todo |
| 16 | A | Frontend setup, theme, API client, shared components | todo |
| 17 | B | Layout shell, campus banner, notification bell | todo |
| 18 | A | Login slideshow, admin login, register, forgot password, route guards | todo |
| 19 | B | Student panel part 1 (routines, exams) | todo |
| 20 | A | Student panel part 2 (notices, materials, directory, polls, teachers, messages) | todo |
| 21 | B | CR panel part 1 (class requests, extra class, room availability, my requests) | todo |
| 22 | A | CR panel part 2 (messaging, notices, materials, polls) | todo |
| 23 | B | Teacher panel | todo |
| 24 | A | Admin panel part 1 (overview, approvals, students, CRs, teachers) | todo |
| 25 | B | Admin panel part 2 (courses, routines, notices, materials) | todo |
| 26 | A | Responsive + animation polish, full manual test | todo |
| 27 | B | Demo seed data, README, API docs, GitHub Actions CI | todo |
