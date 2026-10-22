package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"os"
	"strings"
	"time"
)

const (
	productionEnv  = "production"
	startupTimeout = 60 * time.Second
	demoDomain     = "student.sust.edu"
	staffDomain    = "sust.edu"
)

var (
	errNoFlag     = errors.New("seed: pass -demo to load the demo data")
	errProduction = errors.New("seed: the demo data never loads while APP_ENV is production")
	errNoPassword = errors.New("seed: DEMO_PASSWORD is required in backend/.env")
	errNoPeriods  = errors.New("seed: the class periods are missing, run the migrations first")
)

type teacherSeed struct {
	Identifier  string
	Name        string
	Designation string
	Phone       string
}

type courseSeed struct {
	Code   string
	Title  string
	Credit float64
}

type studentSeed struct {
	Identifier string
	Name       string
	RegNo      string
	Batch      string
	Section    string
	Phone      string
	Blood      string
	IsCR       bool
}

var demoTeachers = []teacherSeed{
	{Identifier: "mrahman", Name: "Mahbubur Rahman", Designation: "Professor", Phone: "01711000001"},
	{Identifier: "fsultana", Name: "Farhana Sultana", Designation: "Associate Professor", Phone: "01711000002"},
	{Identifier: "nhasan", Name: "Nazmul Hasan", Designation: "Assistant Professor", Phone: "01711000003"},
	{Identifier: "achowdhury", Name: "Ayesha Chowdhury", Designation: "Lecturer", Phone: "01711000004"},
	{Identifier: "rkarim", Name: "Rezaul Karim", Designation: "Professor", Phone: "01711000005"},
}

var demoCourses = []courseSeed{
	{Code: "CSE 1101 1101A", Title: "Structured Programming", Credit: 3.00},
	{Code: "CSE 1102 1102B", Title: "Structured Programming Lab", Credit: 1.50},
	{Code: "CSE 2243 7637D", Title: "Digital Logic Design", Credit: 3.00},
	{Code: "CSE 2244 7638E", Title: "Digital Logic Design Lab", Credit: 1.00},
	{Code: "CSE 3131 4411C", Title: "Database Systems", Credit: 3.00},
	{Code: "CSE 3132 4412F", Title: "Database Systems Lab", Credit: 1.50},
	{Code: "CSE 4171 9021G", Title: "Artificial Intelligence", Credit: 3.00},
	{Code: "CSE 4172 9022H", Title: "Software Engineering", Credit: 3.00},
}

var demoBatches = []string{"CSE-21", "CSE-22", "CSE-23", "CSE-24"}

var demoRunning = []string{"4-1", "3-1", "2-1", "1-2"}

var demoOlder = []string{"3-2", "2-2", "1-2", "1-1"}

var demoRooms = []string{"1001", "1002", "1003", "2001"}

var givenNames = []string{
	"Tanvir", "Sadia", "Rakib", "Nusrat", "Shuvo", "Mim", "Arif", "Lamia",
	"Fahim", "Jarin", "Imran", "Rafia", "Sabbir", "Tasnim", "Nayeem", "Sumaiya",
}

var familyNames = []string{
	"Ahmed", "Islam", "Hossain", "Akter", "Chowdhury", "Rahman", "Khan", "Das",
}

var bloodGroups = []string{"A+", "B+", "O+", "AB+", "A-", "B-", "O-", "AB-"}

func main() {
	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func run() error {
	demo := flag.Bool("demo", false, "load the demo data")
	flag.Parse()
	if !*demo {
		return errNoFlag
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if strings.EqualFold(cfg.AppEnv, productionEnv) {
		return errProduction
	}
	password := strings.TrimSpace(os.Getenv("DEMO_PASSWORD"))
	if password == "" {
		return errNoPassword
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	loader := &loader{ctx: ctx, pool: pool, hash: hash, now: cfg.Now()}
	if err := loader.load(); err != nil {
		return err
	}
	loader.report()
	return nil
}

type loader struct {
	ctx       context.Context
	pool      *pgxpool.Pool
	hash      string
	now       time.Time
	periods   []store.Period
	teachers  map[string]int64
	teacherUs map[string]int64
	courses   map[string]int64
	students  []studentSeed
	studentUs map[string]int64
	studentID map[string]int64
	crs       []studentSeed
}

func (l *loader) load() error {
	l.teachers = make(map[string]int64)
	l.teacherUs = make(map[string]int64)
	l.courses = make(map[string]int64)
	l.studentUs = make(map[string]int64)
	l.studentID = make(map[string]int64)
	steps := []func() error{
		l.loadPeriods,
		l.loadCourses,
		l.loadTeachers,
		l.loadStudents,
		l.loadRoutines,
		l.loadItems,
		l.loadExams,
		l.loadPublication,
		l.loadNotices,
		l.loadPolls,
		l.loadConversations,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) loadPeriods() error {
	rows, err := l.pool.Query(l.ctx,
		`SELECT id, label, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), is_break, position
FROM class_periods ORDER BY position`)
	if err != nil {
		return fmt.Errorf("seed: could not read the class periods: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var period store.Period
		err := rows.Scan(&period.ID, &period.Label, &period.Start, &period.End, &period.IsBreak, &period.Position)
		if err != nil {
			return fmt.Errorf("seed: could not scan a class period: %w", err)
		}
		if !period.IsBreak {
			l.periods = append(l.periods, period)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("seed: could not read the class periods: %w", err)
	}
	if len(l.periods) < 4 {
		return errNoPeriods
	}
	return nil
}

func (l *loader) loadCourses() error {
	for _, course := range demoCourses {
		var id int64
		err := l.pool.QueryRow(l.ctx,
			`INSERT INTO courses (code, title, credit) VALUES ($1, $2, $3)
ON CONFLICT (code) DO UPDATE SET title = EXCLUDED.title, credit = EXCLUDED.credit
RETURNING id`, course.Code, course.Title, course.Credit).Scan(&id)
		if err != nil {
			return fmt.Errorf("seed: could not add the course %s: %w", course.Code, err)
		}
		l.courses[course.Code] = id
	}
	return nil
}

func (l *loader) loadTeachers() error {
	for _, teacher := range demoTeachers {
		userID, err := l.upsertUser(teacher.Identifier, "teacher")
		if err != nil {
			return err
		}
		var teacherID int64
		email := teacher.Identifier + "@" + staffDomain
		err = l.pool.QueryRow(l.ctx,
			`INSERT INTO teachers (user_id, teacher_id, name, email, phone, designation)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET name = EXCLUDED.name, designation = EXCLUDED.designation
RETURNING id`, userID, teacher.Identifier, teacher.Name, email, teacher.Phone, teacher.Designation).
			Scan(&teacherID)
		if err != nil {
			return fmt.Errorf("seed: could not add the teacher %s: %w", teacher.Identifier, err)
		}
		l.teachers[teacher.Identifier] = teacherID
		l.teacherUs[teacher.Identifier] = userID
	}
	return nil
}

func (l *loader) loadStudents() error {
	counter := 0
	for batchIndex, batch := range demoBatches {
		year := 2021 + batchIndex
		for _, section := range validate.Sections() {
			for seat := 1; seat <= 4; seat++ {
				name := givenNames[counter%len(givenNames)] + " " + familyNames[counter%len(familyNames)]
				seed := studentSeed{
					Identifier: fmt.Sprintf("s%d%s%d", year%100, strings.ToLower(section), seat),
					Name:       name,
					RegNo:      fmt.Sprintf("%d331%03d", year, counter+1),
					Batch:      batch,
					Section:    section,
					Phone:      fmt.Sprintf("017%08d", 20000000+counter),
					Blood:      bloodGroups[counter%len(bloodGroups)],
					IsCR:       seat == 1,
				}
				counter++
				if err := l.addStudent(seed); err != nil {
					return err
				}
				l.students = append(l.students, seed)
				if seed.IsCR {
					l.crs = append(l.crs, seed)
				}
			}
		}
	}
	return nil
}

func (l *loader) addStudent(seed studentSeed) error {
	userID, err := l.upsertUser(seed.Identifier, "student")
	if err != nil {
		return err
	}
	var studentID int64
	email := seed.RegNo + "@" + demoDomain
	err = l.pool.QueryRow(l.ctx,
		`INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group, is_cr)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET full_name = EXCLUDED.full_name, batch = EXCLUDED.batch,
    section = EXCLUDED.section, is_cr = EXCLUDED.is_cr
RETURNING id`, userID, seed.Name, seed.RegNo, seed.Batch, seed.Section, email, seed.Phone,
		seed.Blood, seed.IsCR).Scan(&studentID)
	if err != nil {
		return fmt.Errorf("seed: could not add the student %s: %w", seed.Identifier, err)
	}
	l.studentUs[seed.Identifier] = userID
	l.studentID[seed.Identifier] = studentID
	return nil
}

func (l *loader) upsertUser(identifier, role string) (int64, error) {
	var userID int64
	err := l.pool.QueryRow(l.ctx,
		`INSERT INTO users (role, identifier, password_hash, status, email_verified_at)
VALUES ($1, $2, $3, 'active', now())
ON CONFLICT (identifier) DO UPDATE SET role = EXCLUDED.role, password_hash = EXCLUDED.password_hash,
    status = 'active'
RETURNING id`, role, identifier, l.hash).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("seed: could not add the account %s: %w", identifier, err)
	}
	return userID, nil
}

func (l *loader) loadRoutines() error {
	for batchIndex, batch := range demoBatches {
		_, err := l.pool.Exec(l.ctx,
			`INSERT INTO batch_semesters (batch, running_semester) VALUES ($1, $2)
ON CONFLICT (batch) DO UPDATE SET running_semester = EXCLUDED.running_semester`,
			batch, demoRunning[batchIndex])
		if err != nil {
			return fmt.Errorf("seed: could not set the running semester of %s: %w", batch, err)
		}
		terms := []struct {
			Semester string
			Slots    int
			Offset   int
		}{
			{Semester: demoRunning[batchIndex], Slots: 3},
			{Semester: demoOlder[batchIndex], Slots: 2, Offset: 3},
		}
		for _, term := range terms {
			if err := l.addRoutine(batch, batchIndex, term.Semester, term.Slots, term.Offset); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *loader) addRoutine(batch string, batchIndex int, semester string, slots, offset int) error {
	for sectionIndex, section := range validate.Sections() {
		for slot := 0; slot < slots; slot++ {
			course := demoCourses[(batchIndex*2+slot+offset)%len(demoCourses)]
			teacher := demoTeachers[(batchIndex+slot+offset)%len(demoTeachers)]
			period := l.periods[(sectionIndex*3+slot+offset)%len(l.periods)]
			room := demoRooms[(batchIndex+slot+offset)%len(demoRooms)]
			kind := "class"
			if strings.Contains(course.Title, "Lab") {
				kind = "lab"
			}
			day := (slot + offset) % 5
			_, err := l.pool.Exec(l.ctx,
				`INSERT INTO routines (batch, semester, section, course_id, teacher_id, day_of_week,
start_time, end_time, room, type)
SELECT $1, $2, $3, $4, $5, $6, $7::time, $8::time, $9, $10
WHERE NOT EXISTS (
    SELECT 1 FROM routines
    WHERE batch = $1 AND semester = $2 AND section = $3 AND day_of_week = $6
      AND start_time = $7::time
)`, batch, semester, section, l.courses[course.Code], l.teachers[teacher.Identifier], day,
				period.Start, period.End, room, kind)
			if err != nil {
				return fmt.Errorf("seed: could not add a routine entry of %s: %w", batch, err)
			}
		}
	}
	return nil
}

func (l *loader) loadItems() error {
	states := []string{
		store.ItemAwaiting,
		store.ItemNoResponse,
		store.ItemConfirmed,
		store.ItemCancelled,
		store.ItemTeacherAsk,
		store.ItemCRAsk,
		store.ItemDeclined,
		store.ItemRescheduled,
		store.ItemScheduled,
	}
	batch := demoBatches[2]
	section := validate.SectionA
	creator := l.studentUs[l.crFor(batch, section).Identifier]
	for index, status := range states {
		course := demoCourses[index%len(demoCourses)]
		teacher := demoTeachers[index%len(demoTeachers)]
		period := l.periods[index%len(l.periods)]
		date := validate.DateString(l.now.AddDate(0, 0, index+1))
		mode, room, link := "room", demoRooms[index%len(demoRooms)], ""
		if status == store.ItemScheduled {
			mode, room, link = "online", "", ""
		}
		note := "Demo item in the " + status + " state"
		if err := l.addItem(batch, section, course.Code, teacher.Identifier, "class", "cr",
			date, period, mode, room, link, status, note, creator); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) addItem(batch, section, courseCode, teacherKey, kind, origin, date string,
	period store.Period, mode, room, link, status, note string, creator int64) error {
	_, err := l.pool.Exec(l.ctx,
		`INSERT INTO schedule_items (batch, section, course_id, teacher_id, kind, origin, item_date,
start_time, end_time, mode, room, meeting_link, status, note, created_by)
SELECT $1, $2, $3, $4, $5, $6, $7::date, $8::time, $9::time, $10, $11, $12, $13, $14, NULLIF($15::bigint, 0)
WHERE NOT EXISTS (
    SELECT 1 FROM schedule_items
    WHERE batch = $1 AND section = $2 AND item_date = $7::date AND start_time = $8::time
      AND course_id = $3
)`, batch, section, l.courses[courseCode], l.teachers[teacherKey], kind, origin, date,
		period.Start, period.End, mode, room, link, status, note, creator)
	if err != nil {
		return fmt.Errorf("seed: could not add a schedule item of %s: %w", batch, err)
	}
	return nil
}

func (l *loader) loadExams() error {
	for index, batch := range demoBatches {
		course := demoCourses[(index*3)%len(demoCourses)]
		teacher := demoTeachers[index%len(demoTeachers)]
		period := l.periods[(index+1)%len(l.periods)]
		date := validate.DateString(l.now.AddDate(0, 0, 10+index))
		_, err := l.pool.Exec(l.ctx,
			`INSERT INTO schedule_items (batch, section, course_id, teacher_id, kind, origin, item_date,
start_time, end_time, mode, room, status, syllabus, instructions, created_by)
SELECT $1, 'ALL', $2, $3, 'exam', 'teacher_exam', $4::date, $5::time, $6::time, 'room', $7,
'scheduled', $8, $9, $10
WHERE NOT EXISTS (
    SELECT 1 FROM schedule_items
    WHERE batch = $1 AND item_date = $4::date AND course_id = $2 AND kind = 'exam'
)`, batch, l.courses[course.Code], l.teachers[teacher.Identifier], date, period.Start, period.End,
			demoRooms[index%len(demoRooms)], "Chapters 1 to 4 of "+course.Title,
			"Bring your student card. Calculators are not allowed.", l.teacherUs[teacher.Identifier])
		if err != nil {
			return fmt.Errorf("seed: could not add an exam of %s: %w", batch, err)
		}
	}
	return nil
}

func (l *loader) loadPublication() error {
	batch := demoBatches[2]
	cr := l.crFor(batch, validate.SectionA)
	var publicationID int64
	err := l.pool.QueryRow(l.ctx,
		`SELECT id FROM publications WHERE batch = $1 ORDER BY published_at DESC LIMIT 1`, batch).
		Scan(&publicationID)
	if err != nil {
		err = l.pool.QueryRow(l.ctx,
			`INSERT INTO publications (batch, published_by) VALUES ($1, $2) RETURNING id`,
			batch, l.studentUs[cr.Identifier]).Scan(&publicationID)
		if err != nil {
			return fmt.Errorf("seed: could not publish the final routine of %s: %w", batch, err)
		}
	}
	_, err = l.pool.Exec(l.ctx,
		`UPDATE schedule_items SET in_final = TRUE
WHERE batch = $1 AND status IN ('confirmed', 'scheduled', 'rescheduled')`, batch)
	if err != nil {
		return fmt.Errorf("seed: could not mark the final routine of %s: %w", batch, err)
	}
	_, err = l.pool.Exec(l.ctx,
		`INSERT INTO published_items (item_id, publication_id, batch, section, item_date, start_time,
end_time, course_code, course_title, teacher_name, kind, mode, room, meeting_link, note,
syllabus, instructions, status)
SELECT i.id, $2, i.batch, i.section, i.item_date, i.start_time, i.end_time, c.code, c.title,
COALESCE(t.name, ''), i.kind, i.mode, i.room, i.meeting_link, i.note, i.syllabus, i.instructions, i.status
FROM schedule_items i
JOIN courses c ON c.id = i.course_id
LEFT JOIN teachers t ON t.id = i.teacher_id
WHERE i.batch = $1 AND i.in_final
ON CONFLICT (item_id) DO NOTHING`, batch, publicationID)
	if err != nil {
		return fmt.Errorf("seed: could not fill the published routine of %s: %w", batch, err)
	}
	return nil
}

func (l *loader) loadNotices() error {
	adminID, err := l.adminUser()
	if err != nil {
		return err
	}
	if adminID > 0 {
		err = l.addNotice("admin", "Mid term schedule is out",
			"The mid term routine for every batch is published on the department board.", "", "", adminID)
		if err != nil {
			return err
		}
	}
	for _, cr := range l.crs[:2] {
		err := l.addNotice("cr", "Class representative notice for "+cr.Batch,
			"Bring the printed lab manual to the next lab session.", cr.Batch, cr.Section,
			l.studentUs[cr.Identifier])
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) addNotice(source, title, description, batch, section string, creator int64) error {
	_, err := l.pool.Exec(l.ctx,
		`INSERT INTO notices (source, title, description, batch, section, created_by)
SELECT $1, $2, $3, NULLIF($4::text, ''), NULLIF($5::text, ''), $6
WHERE NOT EXISTS (SELECT 1 FROM notices WHERE title = $2)`,
		source, title, description, batch, section, creator)
	if err != nil {
		return fmt.Errorf("seed: could not add the notice %q: %w", title, err)
	}
	return nil
}

func (l *loader) loadPolls() error {
	polls := []struct {
		Question string
		Scope    string
		Batch    string
		Section  string
		Max      int
		Options  []string
	}{
		{"Which day suits the make up class?", "section", demoBatches[2], validate.SectionA, 1,
			[]string{"Sunday", "Monday", "Tuesday"}},
		{"Pick the topics for the revision class", "batch", demoBatches[2], validate.SectionAll, 2,
			[]string{"Normalisation", "Indexing", "Transactions", "Query plans"}},
		{"Which week works for the department seminar?", "all_batches", validate.BatchAll,
			validate.SectionAll, 1, []string{"First week", "Second week", "Third week"}},
	}
	cr := l.crFor(demoBatches[2], validate.SectionA)
	for _, poll := range polls {
		var pollID int64
		err := l.pool.QueryRow(l.ctx,
			`SELECT id FROM polls WHERE question = $1`, poll.Question).Scan(&pollID)
		if err == nil {
			continue
		}
		err = l.pool.QueryRow(l.ctx,
			`INSERT INTO polls (question, scope, batch, section, max_choices, created_by)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			poll.Question, poll.Scope, poll.Batch, poll.Section, poll.Max,
			l.studentUs[cr.Identifier]).Scan(&pollID)
		if err != nil {
			return fmt.Errorf("seed: could not add the poll %q: %w", poll.Question, err)
		}
		for position, option := range poll.Options {
			_, err := l.pool.Exec(l.ctx,
				`INSERT INTO poll_options (poll_id, option_text, position) VALUES ($1, $2, $3)`,
				pollID, option, position)
			if err != nil {
				return fmt.Errorf("seed: could not add a poll option: %w", err)
			}
		}
		if err := l.addVotes(pollID, poll.Batch, poll.Section); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) addVotes(pollID int64, batch, section string) error {
	_, err := l.pool.Exec(l.ctx,
		`INSERT INTO poll_votes (poll_id, option_id, student_id)
SELECT $1, first_option.id, s.id
FROM students s
CROSS JOIN LATERAL (
    SELECT o.id FROM poll_options o WHERE o.poll_id = $1 ORDER BY o.position LIMIT 1
) first_option
WHERE ($2::text = 'ALL' OR s.batch = $2::text)
  AND ($3::text = 'ALL' OR s.section = $3::text)
ON CONFLICT DO NOTHING`, pollID, batch, section)
	if err != nil {
		return fmt.Errorf("seed: could not add the poll votes: %w", err)
	}
	return nil
}

func (l *loader) loadConversations() error {
	teacher := demoTeachers[0]
	cr := l.crFor(demoBatches[2], validate.SectionA)
	title := demoBatches[2] + " section " + validate.SectionA
	groupID, created, err := l.group(title, l.teacherUs[teacher.Identifier])
	if err != nil {
		return err
	}
	if created {
		for _, student := range l.students {
			if student.Batch != demoBatches[2] || student.Section != validate.SectionA {
				continue
			}
			if err := l.addMember(groupID, l.studentUs[student.Identifier], store.MemberRegular,
				l.teacherUs[teacher.Identifier]); err != nil {
				return err
			}
		}
		err = l.addMessage(groupID, l.teacherUs[teacher.Identifier],
			"Welcome to the course group. Ask anything about the labs here.")
		if err != nil {
			return err
		}
		if err := l.addMessage(groupID, l.studentUs[cr.Identifier], "Thank you sir."); err != nil {
			return err
		}
	}
	directID, fresh, err := l.direct(l.studentUs[cr.Identifier], l.teacherUs[teacher.Identifier])
	if err != nil {
		return err
	}
	if fresh {
		err = l.addMessage(directID, l.studentUs[cr.Identifier],
			"Sir, may we move tomorrow's class one period later?")
		if err != nil {
			return err
		}
		if err := l.addMessage(directID, l.teacherUs[teacher.Identifier], "That works for me."); err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) group(title string, creator int64) (int64, bool, error) {
	var groupID int64
	err := l.pool.QueryRow(l.ctx,
		`SELECT id FROM conversations WHERE kind = 'group' AND title = $1`, title).Scan(&groupID)
	if err == nil {
		return groupID, false, nil
	}
	err = l.pool.QueryRow(l.ctx,
		`INSERT INTO conversations (kind, title, created_by) VALUES ('group', $1, $2) RETURNING id`,
		title, creator).Scan(&groupID)
	if err != nil {
		return 0, false, fmt.Errorf("seed: could not add the group %q: %w", title, err)
	}
	if err := l.addMember(groupID, creator, store.MemberAdmin, creator); err != nil {
		return 0, false, err
	}
	return groupID, true, nil
}

func (l *loader) direct(left, right int64) (int64, bool, error) {
	var conversationID int64
	err := l.pool.QueryRow(l.ctx,
		`SELECT c.id FROM conversations c
JOIN conversation_members a ON a.conversation_id = c.id AND a.user_id = $1
JOIN conversation_members b ON b.conversation_id = c.id AND b.user_id = $2
WHERE c.kind = 'direct' LIMIT 1`, left, right).Scan(&conversationID)
	if err == nil {
		return conversationID, false, nil
	}
	err = l.pool.QueryRow(l.ctx,
		`INSERT INTO conversations (kind, created_by) VALUES ('direct', $1) RETURNING id`, left).
		Scan(&conversationID)
	if err != nil {
		return 0, false, fmt.Errorf("seed: could not add a direct conversation: %w", err)
	}
	for _, member := range []int64{left, right} {
		if err := l.addMember(conversationID, member, store.MemberRegular, left); err != nil {
			return 0, false, err
		}
	}
	return conversationID, true, nil
}

func (l *loader) addMember(conversationID, userID int64, role string, addedBy int64) error {
	_, err := l.pool.Exec(l.ctx,
		`INSERT INTO conversation_members (conversation_id, user_id, role, added_by)
VALUES ($1, $2, $3, $4) ON CONFLICT (conversation_id, user_id) DO NOTHING`,
		conversationID, userID, role, addedBy)
	if err != nil {
		return fmt.Errorf("seed: could not add a conversation member: %w", err)
	}
	return nil
}

func (l *loader) addMessage(conversationID, senderID int64, body string) error {
	_, err := l.pool.Exec(l.ctx,
		`INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)`,
		conversationID, senderID, body)
	if err != nil {
		return fmt.Errorf("seed: could not add a message: %w", err)
	}
	_, err = l.pool.Exec(l.ctx,
		`UPDATE conversations SET last_message_at = now() WHERE id = $1`, conversationID)
	if err != nil {
		return fmt.Errorf("seed: could not touch a conversation: %w", err)
	}
	return nil
}

func (l *loader) adminUser() (int64, error) {
	var adminID int64
	err := l.pool.QueryRow(l.ctx,
		`SELECT id FROM users WHERE role = 'admin' ORDER BY id LIMIT 1`).Scan(&adminID)
	if err != nil {
		return 0, nil
	}
	return adminID, nil
}

func (l *loader) crFor(batch, section string) studentSeed {
	for _, cr := range l.crs {
		if cr.Batch == batch && cr.Section == section {
			return cr
		}
	}
	return studentSeed{}
}

func (l *loader) report() {
	log.Println("seed: the demo data is loaded, every account uses DEMO_PASSWORD from backend/.env")
	log.Println("seed: teachers")
	for _, teacher := range demoTeachers {
		log.Printf("seed:   %s (%s)", teacher.Identifier, teacher.Designation)
	}
	log.Println("seed: class representatives")
	for _, cr := range l.crs {
		log.Printf("seed:   %s (%s section %s)", cr.Identifier, cr.Batch, cr.Section)
	}
	log.Println("seed: students")
	for _, student := range l.students {
		if student.IsCR {
			continue
		}
		log.Printf("seed:   %s (%s section %s)", student.Identifier, student.Batch, student.Section)
	}
}
