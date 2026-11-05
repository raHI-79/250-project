package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/jobs"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
)

// One week of November 2026, seen from Saturday the 7th at ten in the morning:
//
//	Sun  8 teaching day, the fixed routine carries two classes
//	Mon  9 teaching day, one class the representative created and one cancelled class
//	Tue 10 teaching day, one exam the teacher created
//	Wed 11 a holiday the office added
//	Thu 12 teaching day
//	Fri 13 a weekly off day, with an exam by the representative and a makeup class by the teacher
//	Sat 14 a weekly off day the office turned into a working day that follows Sunday
const (
	weekSunday    = "2026-11-08"
	weekMonday    = "2026-11-09"
	weekTuesday   = "2026-11-10"
	weekWednesday = "2026-11-11"
	weekThursday  = "2026-11-12"
	weekFriday    = "2026-11-13"
	weekSaturday  = "2026-11-14"
)

const (
	weekCourse  = `{"code":"CSE 3101","title":"Operating Systems","credit":3}`
	weekSecond  = `{"code":"CSE 3103","title":"Database Systems","credit":3}`
	weekTeacher = `{"name":"Week Teacher","user_id":"T-900","email":"weekteacher@sust.edu",
"phone":"01712349900"}`
	weekCR = `{"full_name":"Week CR","reg_no":"2023331900","batch":"CSE-23","section":"A",
"email":"weekcr@student.sust.edu","phone":"01712349901","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	weekStudent = `{"full_name":"Week Student","reg_no":"2023331901","batch":"CSE-23","section":"A",
"email":"weekstudent@student.sust.edu","phone":"01712349902","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	weekDropper = `{"full_name":"Week Dropper","reg_no":"2023331902","batch":"CSE-23","section":"A",
"email":"weekdropper@student.sust.edu","phone":"01712349903","blood_group":"B+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
)

func weekClock(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, 11, 7, 10, 0, 0, 0, dhakaZone())
}

type weekFixture struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	router       http.Handler
	adminToken   string
	teacherToken string
	crToken      string
	studentToken string
	dropperToken string
	firstCourse  int64
	secondCourse int64
	teacherRowID int64
}

func (w weekFixture) generate(t *testing.T) {
	t.Helper()
	userStore := store.New(w.pool)
	cfg := flowConfigAt(t, weekClock(t))
	values := settings.New(userStore, SettingDefaults(cfg))
	scheduler := jobs.NewScheduler(cfg, userStore, notify.New(userStore)).WithSettings(values)
	if _, err := scheduler.Generate(w.ctx, cfg.Now()); err != nil {
		t.Fatalf("Generate() returned an unexpected error: %v", err)
	}
}

func seedWeek(t *testing.T) weekFixture {
	t.Helper()
	ctx, pool, router := newFlowRouterAt(t, weekClock(t))
	fixture := weekFixture{ctx: ctx, pool: pool, router: router}
	fixture.adminToken = seedAdmin(t, ctx, pool, router)
	first := call(t, router, http.MethodPost, "/api/admin/courses", weekCourse, fixture.adminToken)
	if first.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", first.Code, first.Body.String())
	}
	fixture.firstCourse = int64(number(t, first, "id"))
	second := call(t, router, http.MethodPost, "/api/admin/courses", weekSecond, fixture.adminToken)
	if second.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", second.Code, second.Body.String())
	}
	fixture.secondCourse = int64(number(t, second, "id"))
	invited := teacherInviteCall(t, router, fixture.adminToken, weekTeacher)
	if invited.Code != http.StatusCreated {
		t.Fatalf("invite teacher status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	fixture.teacherRowID = lookupTeacherID(t, router, fixture.adminToken,
		int64(number(t, invited, "id")))
	// A Sunday routine with two classes, so a working Saturday that follows Sunday has
	// something to generate, and one class on the Monday.
	routines := []string{
		routineBody("A", fixture.firstCourse, fixture.teacherRowID, 0, "09:00", "10:30", "3001", "class"),
		routineBody("A", fixture.secondCourse, fixture.teacherRowID, 0, "10:30", "12:00", "3002", "class"),
		routineBody("A", fixture.firstCourse, fixture.teacherRowID, 1, "09:00", "10:30", "3001", "class"),
		routineBody("A", fixture.secondCourse, fixture.teacherRowID, 4, "09:00", "10:30", "3001", "class"),
	}
	for _, routine := range routines {
		rec := call(t, router, http.MethodPost, "/api/admin/routines", routine, fixture.adminToken)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create routine status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
		}
	}
	onboardCR(t, router, fixture.adminToken, weekCR)
	onboardStudent(t, router, fixture.adminToken, weekStudent)
	onboardStudent(t, router, fixture.adminToken, weekDropper, fixture.secondCourse)
	fixture.teacherToken = loginToken(t, router, "T-900")
	fixture.crToken = loginToken(t, router, "2023331900")
	fixture.studentToken = loginToken(t, router, "2023331901")
	fixture.dropperToken = loginToken(t, router, "2023331902")
	return fixture
}

func saveHoliday(t *testing.T, router http.Handler, adminToken, body string) {
	t.Helper()
	rec := call(t, router, http.MethodPut, "/api/admin/calendar/holidays", body, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("save holiday status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
}

func dayOf(t *testing.T, router http.Handler, token, path, date string) *httptest.ResponseRecorder {
	t.Helper()
	rec := call(t, router, http.MethodGet, path+"?date="+date, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s on %s status = %d, want 200 (body = %q)", path, date, rec.Code, rec.Body.String())
	}
	return rec
}

type chipPayload struct {
	Holiday     bool   `json:"holiday"`
	WorkingDay  bool   `json:"working_day"`
	Name        string `json:"name"`
	FollowsName string `json:"follows_weekday_name"`
	Label       string `json:"label"`
}

func chipOf(t *testing.T, rec *httptest.ResponseRecorder) *chipPayload {
	t.Helper()
	var payload struct {
		Kind *chipPayload `json:"day_kind"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Kind
}

func createdItemID(t *testing.T, rec *httptest.ResponseRecorder) int64 {
	t.Helper()
	var payload struct {
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.Item == nil {
		t.Fatalf("the answer carries no item: %v (body = %q)", err, rec.Body.String())
	}
	return itemID(t, payload.Item)
}

func itemIDsOf(t *testing.T, items []map[string]any) []int64 {
	t.Helper()
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, itemID(t, item))
	}
	return ids
}

func TestOneWeekAgreesAcrossEveryView(t *testing.T) {
	fixture := seedWeek(t)
	router, admin := fixture.router, fixture.adminToken

	t.Run("the office turns Wednesday into a holiday and Saturday into a working day", func(t *testing.T) {
		saveHoliday(t, router, admin, `{"date":"`+weekWednesday+`","name":"Victory Day"}`)
		saveHoliday(t, router, admin,
			`{"date":"`+weekSaturday+`","name":"Make-up Saturday","working_day":true,"follows_weekday":0}`)
		calendar := call(t, router, http.MethodGet, "/api/admin/calendar", "", admin)
		if calendar.Code != http.StatusOK {
			t.Fatalf("calendar status = %d, want 200 (body = %q)", calendar.Code, calendar.Body.String())
		}
	})

	fixture.generate(t)

	t.Run("the generator skips the off days and the holiday and fills the working Saturday", func(t *testing.T) {
		cases := []struct {
			date  string
			count int
			why   string
		}{
			{date: weekSunday, count: 2, why: "the two Sunday classes"},
			{date: weekMonday, count: 1, why: "the one Monday class"},
			{date: weekTuesday, count: 0, why: "no routine on a Tuesday"},
			{date: weekWednesday, count: 0, why: "a holiday"},
			{date: weekThursday, count: 1, why: "the one Thursday class"},
			{date: weekFriday, count: 0, why: "a weekly off day"},
			{date: weekSaturday, count: 2, why: "a working day that follows Sunday"},
		}
		for _, test := range cases {
			rec := dayOf(t, router, fixture.crToken, "/api/cr/upcoming", test.date)
			if got := len(listItems(t, rec)); got != test.count {
				t.Errorf("%s carries %d generated items, want %d (%s)", test.date, got, test.count, test.why)
			}
		}
	})

	t.Run("every view puts the same chip on the holiday", func(t *testing.T) {
		paths := map[string]string{
			"/api/cr/upcoming":               fixture.crToken,
			"/api/cr/final":                  fixture.crToken,
			"/api/teacher/upcoming":          fixture.teacherToken,
			"/api/teacher/upcoming/final":    fixture.teacherToken,
			"/api/students/upcoming-routine": fixture.studentToken,
		}
		for path, token := range paths {
			chip := chipOf(t, dayOf(t, router, token, path, weekWednesday))
			if chip == nil {
				t.Errorf("%s shows no chip on the holiday", path)
				continue
			}
			if !chip.Holiday || chip.Name != "Victory Day" {
				t.Errorf("%s shows %+v on the holiday, want the named holiday", path, chip)
			}
			if chip.Label != "Holiday · Victory Day" {
				t.Errorf("%s labels the holiday %q", path, chip.Label)
			}
		}
	})

	t.Run("every view names the weekly off day and the working day the same way", func(t *testing.T) {
		friday := chipOf(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekFriday))
		if friday == nil || !friday.Holiday || friday.Label != "Holiday · Friday" {
			t.Errorf("the Friday chip = %+v, want the weekly off day", friday)
		}
		saturday := chipOf(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekSaturday))
		if saturday == nil || !saturday.WorkingDay || saturday.FollowsName != "Sunday" {
			t.Errorf("the Saturday chip = %+v, want a working day that follows Sunday", saturday)
		}
		if saturday != nil && saturday.Label != "Working day · follows Sunday" {
			t.Errorf("the Saturday label = %q", saturday.Label)
		}
		student := chipOf(t, dayOf(t, router, fixture.studentToken,
			"/api/students/upcoming-routine", weekSaturday))
		if student == nil || !student.WorkingDay {
			t.Errorf("the student sees %+v on the working Saturday", student)
		}
		teaching := chipOf(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekSunday))
		if teaching != nil {
			t.Errorf("an ordinary teaching day carries the chip %+v, want none", teaching)
		}
	})

	var fridayExam, fridayMakeup, mondayClass, tuesdayExam int64

	t.Run("a representative puts an exam on the Friday off day", func(t *testing.T) {
		body := `{"course_id":` + strconv.FormatInt(fixture.firstCourse, 10) +
			`,"section":"A","kind":"exam","item_date":"` + weekFriday +
			`","start_time":"09:00","end_time":"11:00","mode":"room","room":"3005"}`
		rec := call(t, router, http.MethodPost, "/api/cr/items", body, fixture.crToken)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create on an off day status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
		}
		fridayExam = createdItemID(t, rec)
	})

	t.Run("a teacher asks to move a class and it becomes a makeup class on the Friday", func(t *testing.T) {
		thursday := listItems(t, dayOf(t, router, fixture.teacherToken,
			"/api/teacher/upcoming", weekThursday))
		if len(thursday) != 1 {
			t.Fatalf("the Thursday carries %d items, want the one fixed class", len(thursday))
		}
		fridayMakeup = itemID(t, thursday[0])
		asked := call(t, router, http.MethodPost,
			"/api/teacher/items/"+strconv.FormatInt(fridayMakeup, 10)+"/request-reschedule",
			`{"message":"Let us hold it on the Friday instead"}`, fixture.teacherToken)
		if asked.Code != http.StatusOK {
			t.Fatalf("request reschedule status = %d, want 200 (body = %q)", asked.Code, asked.Body.String())
		}
		moved := call(t, router, http.MethodPatch,
			"/api/cr/items/"+strconv.FormatInt(fridayMakeup, 10),
			`{"item_date":"`+weekFriday+`","start_time":"14:00","end_time":"15:30",`+
				`"mode":"room","room":"3006"}`, fixture.crToken)
		if moved.Code != http.StatusOK {
			t.Fatalf("set the new slot status = %d, want 200 (body = %q)", moved.Code, moved.Body.String())
		}
	})

	t.Run("the Friday off day lists both of them under the chip with their actions", func(t *testing.T) {
		rec := dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekFriday)
		chip := chipOf(t, rec)
		if chip == nil || !chip.Holiday {
			t.Fatalf("the Friday lost its chip: %+v", chip)
		}
		items := listItems(t, rec)
		if len(items) != 2 {
			t.Fatalf("the Friday carries %d items, want the exam and the makeup class", len(items))
		}
		for _, item := range items {
			raw, ok := item["allowed_actions"].([]any)
			if !ok || len(raw) == 0 {
				t.Errorf("an item on the off day lost its actions: %v", item)
			}
		}
		ids := itemIDsOf(t, items)
		seen := map[int64]bool{}
		for _, id := range ids {
			seen[id] = true
		}
		if !seen[fridayExam] || !seen[fridayMakeup] {
			t.Errorf("the Friday carries %v, want %d and %d", ids, fridayExam, fridayMakeup)
		}
		teacher := dayOf(t, router, fixture.teacherToken, "/api/teacher/upcoming", weekFriday)
		if got := len(listItems(t, teacher)); got != 2 {
			t.Errorf("the teacher sees %d items on the off day, want 2", got)
		}
	})

	t.Run("a class on a teaching day is created, cancelled and rescheduled", func(t *testing.T) {
		body := `{"course_id":` + strconv.FormatInt(fixture.secondCourse, 10) +
			`,"section":"A","kind":"class","item_date":"` + weekMonday +
			`","start_time":"14:00","end_time":"15:30","mode":"room","room":"3007"}`
		rec := call(t, router, http.MethodPost, "/api/cr/items", body, fixture.crToken)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
		}
		mondayClass = createdItemID(t, rec)
		generated := listItems(t, dayOf(t, router, fixture.teacherToken,
			"/api/teacher/upcoming", weekMonday))
		if len(generated) != 2 {
			t.Fatalf("the Monday carries %d items, want the fixed class and the new one", len(generated))
		}
		fixed := int64(0)
		for _, item := range generated {
			if item["origin"] == string(store.OriginFixed) {
				fixed = itemID(t, item)
			}
		}
		if fixed == 0 {
			t.Fatal("the Monday carries no fixed class to cancel")
		}
		cancelled := call(t, router, http.MethodPost,
			"/api/teacher/items/"+strconv.FormatInt(fixed, 10)+"/cancel",
			`{"reason":"The teacher is away"}`, fixture.teacherToken)
		if cancelled.Code != http.StatusOK {
			t.Fatalf("cancel status = %d, want 200 (body = %q)", cancelled.Code, cancelled.Body.String())
		}
	})

	t.Run("a teacher puts an exam on the Tuesday and the representative sets its time", func(t *testing.T) {
		body := `{"course_id":` + strconv.FormatInt(fixture.firstCourse, 10) +
			`,"batch":"CSE-23","section":"A","item_date":"` + weekTuesday + `"}`
		rec := call(t, router, http.MethodPost, "/api/teacher/exams", body, fixture.teacherToken)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create exam status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
		}
		tuesdayExam = createdItemID(t, rec)
		items := listItems(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekTuesday))
		if len(items) != 1 {
			t.Fatalf("the Tuesday carries %d items, want the exam", len(items))
		}
		if actions, ok := items[0]["allowed_actions"].([]any); ok {
			found := false
			for _, action := range actions {
				if action == "set_time" {
					found = true
				}
			}
			if !found {
				t.Errorf("the representative is offered %v on the exam, want set_time", actions)
			}
		}
	})

	t.Run("the final routine takes what the representative added and nothing else", func(t *testing.T) {
		for _, id := range []int64{fridayExam, fridayMakeup, mondayClass, tuesdayExam} {
			addToFinal(t, router, fixture.crToken, id)
		}
		friday := dayOf(t, router, fixture.crToken, "/api/cr/final", weekFriday)
		ids := itemIDsOf(t, listItems(t, friday))
		if len(ids) != 2 {
			t.Fatalf("the final Friday carries %v, want both items of that day", ids)
		}
		holiday := dayOf(t, router, fixture.crToken, "/api/cr/final", weekWednesday)
		if got := len(listItems(t, holiday)); got != 0 {
			t.Errorf("the final holiday carries %d items, want none", got)
		}
		if chip := chipOf(t, holiday); chip == nil || !chip.Holiday {
			t.Errorf("the final routine lost the holiday chip: %+v", chip)
		}
	})

	t.Run("a published day reaches the students, the dropper and the home card", func(t *testing.T) {
		published := publishItems(t, router, fixture.crToken, fridayExam, fridayMakeup)
		if published.Code != http.StatusOK {
			t.Fatalf("publish status = %d, want 200 (body = %q)", published.Code, published.Body.String())
		}
		student := dayOf(t, router, fixture.studentToken, "/api/students/upcoming-routine", weekFriday)
		if chip := chipOf(t, student); chip == nil || !chip.Holiday {
			t.Errorf("the student lost the off day chip on the Friday: %+v", chip)
		}
		var payload struct {
			Groups []struct {
				Items []map[string]any `json:"items"`
			} `json:"groups"`
		}
		if err := json.Unmarshal(student.Body.Bytes(), &payload); err != nil {
			t.Fatalf("body is not valid JSON: %v (body = %q)", err, student.Body.String())
		}
		counted := 0
		for _, group := range payload.Groups {
			counted += len(group.Items)
		}
		if counted != 2 {
			t.Errorf("the student sees %d items on the published off day, want 2", counted)
		}
		// A drop course is one the student retakes with another batch, so it adds rows
		// from that batch rather than taking rows away from this one. The student who
		// carries a drop course therefore sees the same day as every other student of
		// the batch.
		dropper := dayOf(t, router, fixture.dropperToken, "/api/students/upcoming-routine", weekFriday)
		var dropped struct {
			Groups []struct {
				Items []map[string]any `json:"items"`
			} `json:"groups"`
		}
		if err := json.Unmarshal(dropper.Body.Bytes(), &dropped); err != nil {
			t.Fatalf("body is not valid JSON: %v (body = %q)", err, dropper.Body.String())
		}
		shown := 0
		for _, group := range dropped.Groups {
			shown += len(group.Items)
		}
		if shown != counted {
			t.Errorf("the student with a drop course sees %d items, want the %d of the batch",
				shown, counted)
		}
		if chip := chipOf(t, dropper); chip == nil || !chip.Holiday {
			t.Errorf("the student with a drop course lost the off day chip: %+v", chip)
		}
	})

	t.Run("a holiday added later clears only the undecided generated items", func(t *testing.T) {
		before := listItems(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekSaturday))
		if len(before) != 2 {
			t.Fatalf("the working Saturday carries %d items, want the two Sunday classes", len(before))
		}
		saveHoliday(t, router, admin, `{"date":"`+weekSaturday+`","name":"Called off"}`)
		after := listItems(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekSaturday))
		if len(after) != 0 {
			t.Errorf("the called-off Saturday still carries %v", itemIDsOf(t, after))
		}
		chip := chipOf(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekSaturday))
		if chip == nil || !chip.Holiday || chip.Name != "Called off" {
			t.Errorf("the Saturday chip = %+v, want the new holiday", chip)
		}
		// The Friday items were created by hand, so the holiday there leaves them alone.
		saveHoliday(t, router, admin, `{"date":"`+weekFriday+`","name":"Also called off"}`)
		kept := listItems(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", weekFriday))
		if len(kept) != 2 {
			t.Errorf("the Friday lost the items somebody created: %v", itemIDsOf(t, kept))
		}
	})

	t.Run("the teacher and the representative see the same days", func(t *testing.T) {
		for _, date := range []string{weekSunday, weekMonday, weekTuesday, weekWednesday,
			weekThursday, weekFriday, weekSaturday} {
			crChip := chipOf(t, dayOf(t, router, fixture.crToken, "/api/cr/upcoming", date))
			teacherChip := chipOf(t, dayOf(t, router, fixture.teacherToken, "/api/teacher/upcoming", date))
			if (crChip == nil) != (teacherChip == nil) {
				t.Errorf("%s: the representative sees %+v and the teacher %+v", date, crChip, teacherChip)
				continue
			}
			if crChip != nil && crChip.Label != teacherChip.Label {
				t.Errorf("%s: the chips disagree, %q and %q", date, crChip.Label, teacherChip.Label)
			}
		}
	})
}
