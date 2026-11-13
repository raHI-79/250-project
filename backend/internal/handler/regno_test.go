package handler

import (
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"strings"
	"testing"
)

func registerWith(t *testing.T, values map[string][]string) (*accountStore, int, string) {
	t.Helper()
	users := newAccountStore()
	handler := newAccountAuth(t, users, testPhotos(t), &fakeGoogle{}, &fakeMailer{})
	request := multipartRequest(t, http.MethodPost, "/api/auth/register", values)
	rec := serve(handler.Register(), request)
	message := ""
	if rec.Code != http.StatusCreated {
		message = errorMessage(t, rec)
	}
	return users, rec.Code, message
}

func registrationFor(regNo string) map[string][]string {
	values := withField(registrationForm(), "reg_no", regNo)
	return withField(values, "email", regNo+"@student.sust.edu")
}

func TestRegisterDerivesTheBatchFromTheRegistrationNumber(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name      string
		regNo     string
		sent      string
		wantBatch string
	}{
		{name: "2021 is CSE-21", regNo: "2021331001", sent: "CSE-21", wantBatch: "CSE-21"},
		{name: "2023 is CSE-23", regNo: "2023331079", sent: "CSE-23", wantBatch: "CSE-23"},
		{name: "2025 with twelve digits is CSE-25", regNo: "202533107912", sent: "CSE-25", wantBatch: "CSE-25"},
		{name: "a different batch from the client is ignored", regNo: "2023331079", sent: "CSE-21",
			wantBatch: "CSE-23"},
		{name: "no batch from the client", regNo: "2025331001", sent: "", wantBatch: "CSE-25"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := registrationFor(test.regNo)
			if test.sent == "" {
				delete(values, "batch")
			} else {
				values = withField(values, "batch", test.sent)
			}
			users, code, message := registerWith(t, values)
			if code != http.StatusCreated {
				t.Fatalf("status = %d, want 201 (error = %q)", code, message)
			}
			if got := users.registrations[0].Batch; got != test.wantBatch {
				t.Errorf("stored batch = %q, want %q", got, test.wantBatch)
			}
			if got := users.registrations[0].RegNo; got != test.regNo {
				t.Errorf("stored registration number = %q, want %q", got, test.regNo)
			}
		})
	}
}

func TestRegisterRefusesAnUnknownOrInactiveBatch(t *testing.T) {
	silenceLog(t)
	batches := newFakeBatchStore()
	batches.rows["CSE-21"] = store.Batch{Name: "CSE-21", Status: store.BatchInactive}
	UseBatchGate(batches)
	t.Cleanup(func() { UseBatchGate(nil) })
	tests := []struct {
		name      string
		regNo     string
		wantCode  int
		wantError string
	}{
		{name: "an active batch", regNo: "2023331079", wantCode: http.StatusCreated},
		{name: "an unknown batch", regNo: "2025331001", wantCode: http.StatusBadRequest,
			wantError: "CSE-25 is not open for registration. Please contact the department office."},
		{name: "an inactive batch", regNo: "2021331001", wantCode: http.StatusBadRequest,
			wantError: "CSE-21 is not open for registration. Please contact the department office."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users, code, message := registerWith(t, registrationFor(test.regNo))
			if code != test.wantCode || message != test.wantError {
				t.Fatalf("status = %d with %q, want %d with %q", code, message, test.wantCode, test.wantError)
			}
			if test.wantError != "" && len(users.registrations) != 0 {
				t.Errorf("registrations = %+v, want nothing stored", users.registrations)
			}
		})
	}
}

func TestRegisterAcceptsAnyLengthOfRegistrationNumber(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name      string
		regNo     string
		wantError string
	}{
		{name: "six digits", regNo: "202301"},
		{name: "ten digits", regNo: "2023331079"},
		{name: "twelve digits", regNo: "202333107912"},
		{name: "twenty digits", regNo: "20233310791234567890"},
		{name: "five digits", regNo: "20233",
			wantError: "Registration number must start with 20 and be 6 to 20 digits long."},
		{name: "twenty-one digits", regNo: "202333107912345678901",
			wantError: "Registration number must start with 20 and be 6 to 20 digits long."},
		{name: "letters", regNo: "2023A31079", wantError: "Registration number may only use digits."},
		{name: "a dash", regNo: "2023-331079", wantError: "Registration number may only use digits."},
		{name: "another century", regNo: "1923331079",
			wantError: "Registration number must start with 20 and be 6 to 20 digits long."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, code, message := registerWith(t, registrationFor(test.regNo))
			if test.wantError == "" {
				if code != http.StatusCreated {
					t.Fatalf("status = %d, want 201 (error = %q)", code, message)
				}
				return
			}
			if code != http.StatusBadRequest || message != test.wantError {
				t.Errorf("status = %d with %q, want 400 with %q", code, message, test.wantError)
			}
		})
	}
}

func TestSignInAndResetWorkForTenAndTwelveDigits(t *testing.T) {
	silenceLog(t)
	for _, regNo := range []string{"2023331001", "202333100112"} {
		t.Run(regNo, func(t *testing.T) {
			users := newAuthStore(t)
			student := users.users["2023331001"]
			student.Identifier = regNo
			users.users[regNo] = student
			users.targets[regNo] = users.targets["2023331001"]
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			signedIn := post(t, handler.Login(), "/api/auth/login",
				`{"identifier":"`+regNo+`","password":"student-pass"}`)
			if signedIn.Code != http.StatusOK {
				t.Fatalf("sign-in status = %d, want 200 (body = %s)", signedIn.Code, signedIn.Body.String())
			}
			reset := post(t, handler.ForgotPassword(), "/api/auth/forgot-password",
				`{"identifier":"`+regNo+`","date_of_birth":"2003-05-14"}`)
			if reset.Code != http.StatusOK {
				t.Fatalf("reset status = %d, want 200 (body = %s)", reset.Code, reset.Body.String())
			}
		})
	}
}

func TestOfficeSeesTheDerivedBatch(t *testing.T) {
	tests := []struct {
		regNo  string
		stored string
		want   string
	}{
		{regNo: "2023331079", stored: "CSE-21", want: "CSE-23"},
		{regNo: "202533107912", stored: "CSE-25", want: "CSE-25"},
		{regNo: "old-number", stored: "CSE-19", want: "CSE-19"},
	}
	for _, test := range tests {
		if got := derivedBatch(test.regNo, test.stored); got != test.want {
			t.Errorf("derivedBatch(%q, %q) = %q, want %q", test.regNo, test.stored, got, test.want)
		}
	}
}

func TestHomeHeaderCarriesNoWeek(t *testing.T) {
	encoded, err := json.Marshal(todayResponse{Date: "2026-11-12", Semester: "3-1"})
	if err != nil {
		t.Fatalf("could not encode the home response: %v", err)
	}
	if strings.Contains(string(encoded), `"week"`) {
		t.Errorf("home response = %s, want no week", encoded)
	}
	if !strings.Contains(string(encoded), `"date":"2026-11-12"`) {
		t.Errorf("home response = %s, want the date kept", encoded)
	}
}
