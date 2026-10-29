# Development Roadmap

The project is built in 24 development steps. Each step is owned by one team
member, is worked on alone, and ends with reviewed Git commits. The last step is
shared by both members.

- **Owner A** — Ratul Islam ([@raHI-79](https://github.com/raHI-79))
- **Owner B** — Rakibul Hasan ([@niloySapiens](https://github.com/niloySapiens))

Status values: `todo` → `in progress` → `done`.

Full requirements: [SPEC.md](SPEC.md)

## Steps

| Step | Owner | Title | Status |
|---:|:---:|---|---|
| 1 | A | Foundation and Go server skeleton | done |
| 2 | A | PostgreSQL connection and database schema | done |
| 3 | A | Security core: bcrypt, JWT, middleware, admin seed | done |
| 4 | B | Registration, login, logout, password reset | done |
| 5 | A | Admin: approvals, students, CRs, teachers, stats | done |
| 6 | B | Courses, fixed routines, lookups, notification service | done |
| 7 | A | CR class/exam/extra requests and room availability | done |
| 8 | B | Teacher approvals, cancel, schedule, auto-cancel, student routine APIs | done |
| 9 | A | Uploads, notices, materials, directory, teachers info | done |
| 10 | B | Messages, polls, security tests | done |
| 11 | A | Frontend foundation: theme, components, layouts, notifications | done |
| 12 | B | Auth pages with email code password reset, student panel | done |
| 13 | B | CR panel | done |
| 14 | A | Admin panel | done |
| 15 | B | Teacher panel | done |
| 16 | B | Account backend: verification, invitations, promotion, profiles | done |
| 17 | A | Account and profile pages | done |
| 18 | B | Unified messaging, teacher notices and attachments | done |
| 19 | A | Routine experience, batch Drive links and CR student access | done |
| 20 | B | Teacher-chosen user IDs, open teacher email and admin record retention | done |
| 21 | A | Scheduling engine, notification links, course codes and verification paths | done |
| 22 | B | Scheduling pages, messaging search and clickable notifications | done |
| 23 | A | Interface redesign: design system, public home, sign-in and dashboard pages | done |
| 24 | A+B | Polls, messaging rules, group chats, push notifications, class periods, sections, course credits, demo data, docs and CI, semesters, course list and chat delete | done |
| 25 | A+B | Release 2 part one: push delivery, Apps Script mail, sign-in and readability, photo cropping, section badges and teacher reminders, owner edits and drop courses, versioning and the human check | done |
| 26 | A+B | Native Android app on Capacitor with Firebase notifications, a signed APK from GitHub Actions and website-only office access | done |
