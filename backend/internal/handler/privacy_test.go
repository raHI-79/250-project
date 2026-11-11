package handler

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/store"
)

var privateFields = []string{"date_of_birth", "token_key"}

var officeBodies = map[string]bool{
	"studentItem":      true,
	"teacherItem":      true,
	"registrationItem": true,
	"personDetail":     true,
	"profileResponse":  true,
	"sessionBody":      true,
}

func TestAPrivateFieldOnlyAppearsWhereItBelongs(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("could not list the handler sources: %v", err)
	}
	strays := make([]string, 0)
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("could not read %s: %v", name, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			t.Fatalf("could not parse %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structure, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structure.Fields.List {
				if field.Tag == nil {
					continue
				}
				for _, private := range privateFields {
					if !strings.Contains(field.Tag.Value, `json:"`+private) {
						continue
					}
					if officeBodies[spec.Name.Name] {
						continue
					}
					if strings.HasSuffix(spec.Name.Name, "Request") {
						continue
					}
					strays = append(strays,
						name+": "+spec.Name.Name+" carries "+private)
				}
			}
			return true
		})
	}
	if len(strays) > 0 {
		t.Errorf("a private field reached a response shape nobody vouched for: %v", strays)
	}
}

func TestTheAttendanceKeyNeverReachesAStudent(t *testing.T) {
	now := time.Now()
	session := store.AttendanceSession{
		ID:        4,
		TokenKey:  "the-shared-secret",
		Status:    attendance.StatusOpen,
		Method:    attendance.TakeBluetooth,
		StartedAt: now.Add(-time.Minute),
		EndsAt:    now.Add(9 * time.Minute),
	}
	student := newSessionBody(session, now, false)
	if student.TokenKey != "" {
		t.Errorf("a student received the token key %q", student.TokenKey)
	}
	if student.Code != "" {
		t.Errorf("a student received the code %q", student.Code)
	}
	raw, err := json.Marshal(student)
	if err != nil {
		t.Fatalf("could not encode the session: %v", err)
	}
	if strings.Contains(string(raw), session.TokenKey) {
		t.Errorf("the key reached the body of a student: %s", raw)
	}
	teacher := newSessionBody(session, now, true)
	if teacher.TokenKey != session.TokenKey {
		t.Errorf("the teacher lost the key it needs to show the code: %q", teacher.TokenKey)
	}
	closed := session
	closed.Status = attendance.StatusClosed
	if body := newSessionBody(closed, now, true); body.TokenKey != "" {
		t.Errorf("a closed session still carried the key %q", body.TokenKey)
	}
	roll := session
	roll.Method = attendance.TakeRollCall
	if body := newSessionBody(roll, now, true); body.TokenKey != "" {
		t.Errorf("a roll call carried the key %q", body.TokenKey)
	}
}

func TestNoErrorMessageCarriesASecret(t *testing.T) {
	forbidden := []string{
		"password_hash", "reset_token", "token_key", "JWT_SECRET", "DATABASE_URL",
		"VAPID_PRIVATE_KEY", "FIREBASE_SERVICE_ACCOUNT", "Bearer ", "postgres://",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("could not list the handler sources: %v", err)
	}
	leaks := make([]string, 0)
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("could not read %s: %v", name, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			t.Fatalf("could not parse %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			spec, ok := node.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for index, ident := range spec.Names {
				if !strings.HasPrefix(ident.Name, "Message") || index >= len(spec.Values) {
					continue
				}
				literal, ok := spec.Values[index].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				for _, secret := range forbidden {
					if strings.Contains(literal.Value, secret) {
						leaks = append(leaks, name+": "+ident.Name)
					}
				}
			}
			return true
		})
	}
	if len(leaks) > 0 {
		t.Errorf("a message carries something a caller should not see: %v", leaks)
	}
}
