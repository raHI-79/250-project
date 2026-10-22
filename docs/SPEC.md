# SUST CSE Class & Exam Coordination System

> Project specification for the CSE 250 course project.
> A complete, professional, responsive, database-driven web application for the SUST CSE department.

---

## 0. Note for Developers (read first)

| Rule | Details |
|---|---|
| **MUST follow exactly** | All features and access rules for **Student, CR, Teacher and Administration** (sections 2 to 9). Do not remove or change what each role can and cannot do. |
| **use** | **Go (Golang)** for the backend and **PostgreSQL** for the database. |
| **Flexible** | UI/UX and design (section 10) are guidance, not strict rules. Better choices are welcome as long as the app stays professional, responsive (mobile friendly), light-themed and easy to use. |
| **Flexible** | Technical details such as table column names, API endpoint names, libraries and folder structure may be improved if there is a better professional approach, as long as every feature works the same for the user. |

---

## 1. Overview

### 1.1 Purpose
Help the SUST CSE department coordinate classes, exams, notices, materials and communication between students, Class Representatives (CRs), teachers and the department administration.

### 1.2 Tech Stack
| Layer | Technology |
|---|---|
| Backend | Go (Golang), REST JSON API |
| Database | PostgreSQL |
| Authentication | bcrypt password hashing + JWT |
| Frontend | React + Vite (flexible) |
| File uploads | PDF files stored on the server; file path saved in the database |

### 1.3 Roles
| Role | Description |
|---|---|
| **Student** | Views routines, exams, notices, materials, polls, directory and messages. Cannot send messages. |
| **CR** (Class Representative) | A student with extra powers for **their own batch and section**. |
| **Teacher** | Approves class requests, manages schedule, communicates with CRs and students. |
| **Administration** | Hidden admin panel. Manages users, courses, routines, notices and materials. |

---

## 2. Roles & Access Control (CRITICAL)

### 2.1 System parts
- The system has **two parts**:
  1. **Shared part** for Student, CR and Teacher.
  2. **Hidden admin part.**
- Normal users must **never** see any admin link, menu, button or page.
- Admin logs in at a separate, **unlinked** route: `/office-portal` (not linked anywhere in the normal UI).

### 2.2 Backend enforcement
- The backend must verify the role on **every** protected route using middleware (`verifyToken` + `requireRole`).
- **Never** rely on hiding frontend buttons for security.
- A Student, CR or Teacher token calling any `/api/admin/*` route must receive **403** with the message `403 Access Denied / Unauthorized Access`.
- A missing or invalid token must receive **401**.

### 2.3 CR role rules
- A CR is stored as a **student record with `is_cr = true`**.
- The JWT **must** carry the **effective role** (`cr` or `student`), not the raw database role. Otherwise CR-only routes would reject the CR.
- **CR-only routes** allow only role `cr`.
- **Shared student routes** (`/api/students/*`) allow both `student` and `cr`.
- A CR can only control **their own batch + section**. Every CR query must be scoped by the authenticated CR's batch and section from their profile.

### 2.4 Messaging restriction
- Students can only **receive** messages from teachers and CRs.
- Students have **no ability** to send messages to teachers: **no UI and no API**.

### 2.5 Login & account security
- **Unified login** for Student, CR and Teacher:
  - Student / CR → **Registration Number** + password
  - Teacher → **Teacher ID** + password
- **Separate admin login** endpoint.
- Secure **logout**.
- **Password reset** flow (token based). For the demo, return the reset token in the API response. In production it should be emailed.

---

## 3. Database Tables

All tables must use proper **foreign keys**.

| Table | Main fields |
|---|---|
| **users** | role, identifier, password_hash, status (`active` / `pending` / `rejected` / `removed`), reset_token |
| **students** | full_name, reg_no, batch, section, email, phone, blood_group, has_drop_course, drop_course_code, is_cr |
| **teachers** | name, teacher_id, email, phone |
| **courses** | code, title |
| **routines** | Fixed weekly routine: batch, section, course, teacher, day_of_week, start_time, end_time, room, type (`class` / `lab`) |
| **class_events** | Dated class/exam/extra class: batch, section, course, teacher, date, start_time, end_time, room, event_type (`class` / `exam` / `extra`), status (`pending` / `approved` / `rejected` / `cancelled` / `auto_cancelled`), created_by |
| **polls** | Poll question and settings |
| **poll_options** | Options for each poll |
| **poll_votes** | One row per vote. **UNIQUE (poll, student)** |
| **notices** | source (`admin` / `cr`), title, description, batch, section, file_path, created_by |
| **materials** | batch (or `ALL`), title, link, created_by |
| **messages** | sender, sender_role, sender_name, target_type, target_batch, target_section, target_user_id, content, file_path |
| **notifications** | recipient_user_id **or** recipient_role / batch / section (broadcast), type, title, body, created_at |

---

## 4. Student Registration & Account Flow

### 4.1 Registration form fields
- Full Name
- Registration Number
- Batch (e.g. `CSE-23`)
- Section
- Email
- Phone
- Blood Group
- New Password
- Confirm Password
- "Do you have any drop course?" → **Yes** (then enter the course code) / **No**

### 4.2 Rules
- New registrations have status **`pending`**. They **cannot log in** until Administration approves them.
- Administration can **approve** or **reject** registrations.
- Clear login messages for **pending**, **rejected** and **removed** accounts.
- Passwords must be hashed with **bcrypt**, never stored as plain text.
- **Forgot / Reset Password** page with 3 steps:
  1. Enter ID
  2. Get reset token
  3. Set new password

---

## 5. Student Panel

Sidebar navigation. **Only** these features:

### A. Upcoming Routine
- Shows the **next day's** classes and exams, merged from the **fixed routine** + **approved dated events**.
- Organized by date, time, batch, section, course, teacher and room.
- Display format example:
  ```
  CSE-23 (2/2) 07.09.26 (Monday)
  Section A:      10:30 AM - 12:45 PM  DLD Lab @ 1003
  Section B:      ...
  Both Sections:  ...
  ```
- **Cancelled classes must be shown as cancelled.**

### B. Drop Course Routine
- If the student answered **Yes** with a course code: show the routine (class + exam) of that course from the **junior batch(es)** that take it, **in addition to** their own batch routine.
- If **No**: show only their own batch routine.

### C. Upcoming Exams / Assignments
- Date, time, course, batch, section, room.
- Approved exam events **from today onward**.

### D. Current Semester Fixed Routine
- Filters: **Day**, **Section**, **Course**, plus **Batch** and **Section** dropdowns.
- Default = the student's own batch and section; the student can browse other batches and sections.
- Lookup endpoints for batches and sections.

### E. CR Notices
- Filter toggle: **"My Batch"** vs **"Other Batches"**.
- Notices from other batches show a **batch badge**.
- Supports a **PDF attachment** link.

### F. Administration Notices
- Title, description, date, optional PDF/file attachment.
- The attachment **must actually save `file_path` to the database** and show a link.

### G. Materials
- Google Drive links.
- A student sees links targeted to **their own batch** plus links marked **ALL** (all batches).

### H. All Batch Students Directory
- Columns: Name, Reg No, Batch, Section, Phone, Email, Blood Group.
- **CR badge** for CRs.
- Filters: **Batch** dropdown, **Blood Group** dropdown, and **text search** (name / reg no).
- Only **active** students are listed.

### I. Polls
- Question, multiple options, voting UI with **result bars**.
- **Open / closed** status.
- **One vote per student per poll**, enforced in **both the database and the API**.

### J. Teachers Info
- List of teachers (Name, Phone, Email, from the Admin's teacher registration).
- Search by name.

### K. Messages (read-only inbox)
- Direct messages to the student, plus batch/section broadcasts from teachers or the CR.
- PDF links.
- **No reply or send UI.**

---

## 6. CR Panel

**Only** these features:

### 6.1 Fixed Routine
- Fixed routine of the CR's **own batch + section**.

### 6.2 Create Class / Exam
- For any date (including the next day). Choose: type, course, teacher, date, time, room.
- The request goes to the selected teacher as **PENDING**.
- It becomes visible to students **only after teacher approval**.
- If **not approved by 11:59 PM the night before** (i.e. still pending when its date arrives), it is **automatically cancelled**. A background job runs every few minutes and sets status to `auto_cancelled`.
- **"My Requests"** shows the status of each request: approved / rejected / pending / auto-cancelled.

### 6.3 Create Extra Class
- Fields: batch, section, course, teacher, date, start time, end time, room.
- **Must show room availability BEFORE submitting.**

### 6.4 Room Availability Panel
- Appears when a date is chosen.
- Lists **everything booked that day** (room, time, batch/section, course, source), combining:
  - one-off **dated bookings**, and
  - the recurring **Fixed Routine** for that weekday.
- Entries from the CR's **own batch's** fixed routine are labelled **"yours"** (green, not blocking).
- Entries from **other batches** are labelled **"unavailable"** (red).
- **First CR to book a free room/time wins**; later requests see it as unavailable.
- The server rejects conflicts with **409** and the message `This room is not available at the selected time.` (or a message naming the other batch's fixed slot).

### 6.5 Class Hours Rule
- Class hours are restricted to **08:00 – 17:00** (8 AM – 5 PM).
- Start time must be **before** end time.
- Enforced on the **backend**, and with min/max on the frontend time inputs.

### 6.6 Messaging
- Send a message and/or PDF to:
  1. **Own batch** group
  2. **Another batch** group
  3. **A specific teacher** (use the teacher's **users** table id, not the teachers table id)
- **Teacher Messages inbox**: messages teachers sent to this CR.
- **Two-way communication** with teachers (CR can message a teacher, teacher can reply).

### 6.7 Post Notice
- Optional PDF.
- Target: **own batch** OR a **specific other batch** (type the batch code).

### 6.8 Materials (Drive links)
- Visibility selector: **"My batch only"**, **"A specific other batch"**, **"All batches"**.
- The CR sees and can **remove** materials they added.

### 6.9 Polls
- Create a poll: question, multiple options, target **own section** or **whole batch**.
- **Close** a poll.

### 6.10 Teachers Info
- Same as the student's Teachers Info tab.

---

## 7. Teacher Panel

**Only** these features:

### 7.1 Daily Schedule
- Pick a date. Shows the **fixed routine** for that weekday + **approved dated events** (date, time, batch, section, course, room).

### 7.2 Approval Requests
- Pending class / exam / extra-class requests naming this teacher.
- Shows batch, section, course, date, time, room, requested by.
- **Approve** or **Reject**.

### 7.3 Cancel Class
- The teacher can cancel their approved classes **at any time**.
- List of all upcoming approved classes with **one-click cancel**, plus **browse by date**.
- **Students and the CR are notified.**

### 7.4 CR Messages
- Inbox of messages from CRs.
- **Reply** with text and/or PDF.

### 7.5 Send Message / Notice
- Optional PDF. Send to:
  - a **specific CR**,
  - a **batch** (whole batch or one section), or
  - **one specific student** (search by name / reg no).
- Lookup endpoints for CRs, batches, sections and student search.

---

## 8. Administration Panel (hidden)

**Only** these features:

### 8.1 Student Registration Approvals
- Approve / reject. Shows **drop course** info.

### 8.2 All Students
- List with search.
- **Remove (deactivate)** any student at any time.

### 8.3 Manage CRs
- **Add**: name, reg no, batch, section, email, phone, blood group, password.
- List, remove / deactivate.
- CR info also appears in the student directory.

### 8.4 Manage Teachers
- **Add**: name, teacher ID, email, phone, password.
- List, remove / deactivate.

### 8.5 Courses
- Add and list.

### 8.6 Fixed Routines
- Add entries: batch, section, course, teacher, day, start time, end time, room, type.
- The backend must **REJECT** a second entry for the **same room + same weekday + overlapping time** with **409**, naming the existing booking.
- Enforce **08:00 – 17:00**.
- Listing has **Batch** and **Section** filter dropdowns (batch-wise and section-wise view).
- Sorted by batch, section, day, time.

### 8.7 Notices
- Publish with optional PDF (and optional batch).
- Delete.

### 8.8 Materials
- Add Drive links per batch, list.

### 8.9 Overview Dashboard
- **Animated count-up** stat cards:
  - Total students
  - CRs
  - Teachers
  - Pending approvals

---

## 9. Notifications (in-app)

### 9.1 UI
- **Bell icon** in the header for Student, CR and Teacher.
- **Unread badge** and a dropdown list.
- **Polls every 30 seconds** ("last seen" tracked in `localStorage`).

### 9.2 When notifications are created
| Event | Who is notified |
|---|---|
| CR or Admin posts a notice | Targeted students |
| Teacher approves / rejects a CR's request | That CR |
| Teacher approves a class | Students of that batch / section |
| Teacher cancels a class | Students and the CR |
| CR creates a class / exam / extra-class request | That teacher |
| CR messages a teacher | That teacher |
| Teacher replies / sends a message to a CR, batch or student | The recipients |

### 9.3 Scope
- **Broadcast** notifications are scoped by **batch / section**.
- **Direct** notifications are scoped by **user id**.

---

## 10. UI / UX / Design (guidance, flexible)

> These are guidelines. Better design choices are welcome as long as the app stays professional, responsive, light-themed and easy to use.

### 10.1 Theme
- Professional **light theme** (not full black).
- Warm off-white page background (`#F7F7F4`), **white cards**.
- **Forest-green** primary colour (brand ramp around `#1f5636`).
- **Brick / terracotta** secondary accent (inspired by the campus photos).
- **Dark forest-green sidebar**, **white top header**.
- **Serif font** only for the wordmark and page titles; **sans-serif** for all dense UI.

### 10.2 Responsive
- Mobile friendly.
- Sidebar collapses into a **hamburger drawer** on small screens.
- Tables **scroll horizontally** on small screens.
- Each role has its **own dashboard**; irrelevant features are not shown.

### 10.3 Shared components
- **Loading skeleton shimmer** (not a plain spinner)
- **Empty states** with an icon
- **Inline alert banners** for form feedback
- **Toast notifications** (auto-dismiss, top-right) for action results
- **Styled confirm dialog** modal for every destructive action (**never** use the native `window.confirm`)
- **Status badges**: pending / approved / rejected / cancelled

### 10.4 Animations
- Tab content **fade-in-up** on every tab switch
- Dropdown **scale-in**
- Alert **fade-in**
- Toast **slide-in**
- Button **press** (active scale)
- Card **hover shadow**
- Sidebar item **hover micro-interaction**
- **Animated count-up** on admin stats
- Login card **entrance animation**

### 10.5 Campus photos
Real SUST campus photos (Shaheed Minar monument, building, campus entrance road), stored in `frontend/src/assets/`:

| File | Size |
|---|---|
| `hero-monument.jpg`, `hero-building.jpg`, `hero-entrance.jpg` | 1920 × 1080 |
| `banner-monument.jpg`, `banner-building.jpg`, `banner-entrance.jpg` | 1600 × 260 |

Usage:
| Page | Background |
|---|---|
| Student / CR / Teacher login | Full-bleed, auto-crossfading slideshow of the 3 hero images (18s loop) with a green-tinted gradient overlay |
| Admin login | A single static, heavily darkened hero image |
| Register | Decorative banner above the form |
| Forgot Password | Soft, white-washed photo background |
| Every dashboard (all 4 roles) | Slim, animated crossfading campus banner strip at the top with the university / department name |

Users must be able to **replace images** by overwriting files with the same names.
