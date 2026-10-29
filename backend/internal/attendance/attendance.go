package attendance

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	TokenSliceSeconds = 10
	CodeSliceSeconds  = 20
	TokenBytes        = 8
	CodeDigits        = 4
	keyBytes          = 32

	MaxDistanceM     = 30.0
	MaxAccuracyM     = 50.0
	TeacherFreshness = 60 * time.Second

	MinDuration = 5 * time.Minute
	MaxDuration = 30 * time.Minute

	AlertPercentage     = 50.0
	AlertMinSessions    = 3
	AlertCooldown       = 7 * 24 * time.Hour
	SemesterChangeLimit = 2

	StatusOpen    = "open"
	StatusClosed  = "closed"
	Present       = "present"
	Absent        = "absent"
	MethodApp     = "app"
	MethodCode    = "code"
	MethodManual  = "manual"
	MethodSystem  = "system"
	SectionAll    = "ALL"
	RequestOpen   = "pending"
	RequestTaken  = "approved"
	RequestTurned = "rejected"
)

const (
	MessageWrongPhone    = "This is not your registered phone. Request to use this phone."
	MessageWeakSignal    = "Weak GPS signal — move near a window and try again."
	MessageMockLocation  = "This position comes from a mock location app. Turn it off and try again."
	MessageNoPosition    = "Your phone did not report a position. Turn location on and try again."
	MessageTeacherStale  = "The teacher's position is out of date. Ask the teacher to keep the app open."
	MessageTooFar        = "You are too far from the class to be marked present."
	MessageWrongToken    = "That code is no longer valid. Use the code on the board."
	MessageSessionClosed = "This attendance is closed."
	MessageAlreadyMarked = "You are already marked for this class."
	MessagePhoneUsed     = "This phone already marked another student in this class."
	MessageNotOnRoster   = "This class is not on your routine."
	MessageWebsiteOnly   = "Attendance is marked from the Android app. Open the app and tap I'm present."
	MessageBadDuration   = "Attendance stays open between 5 and 30 minutes."
)

func NewKey() (string, error) {
	raw := make([]byte, keyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("attendance: could not draw a session key: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func Slice(now time.Time, seconds int) int64 {
	if seconds <= 0 {
		return 0
	}
	return now.Unix() / int64(seconds)
}

func digest(key string, sessionID, slice int64) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	fmt.Fprintf(mac, "%d:%d", sessionID, slice)
	return mac.Sum(nil)
}

func Token(key string, sessionID, slice int64) string {
	return hex.EncodeToString(digest(key, sessionID, slice)[:TokenBytes])
}

func Code(key string, sessionID, slice int64) string {
	sum := digest(key, sessionID, slice)
	value := binary.BigEndian.Uint32(sum[:4]) % 10000
	return fmt.Sprintf("%04d", value)
}

func CurrentToken(key string, sessionID int64, now time.Time) string {
	return Token(key, sessionID, Slice(now, TokenSliceSeconds))
}

func CurrentCode(key string, sessionID int64, now time.Time) string {
	return Code(key, sessionID, Slice(now, CodeSliceSeconds))
}

func TokenMatches(key string, sessionID int64, now time.Time, offered string) bool {
	offered = strings.ToLower(strings.TrimSpace(offered))
	if len(offered) != TokenBytes*2 {
		return false
	}
	slice := Slice(now, TokenSliceSeconds)
	return sameString(Token(key, sessionID, slice), offered) ||
		sameString(Token(key, sessionID, slice-1), offered)
}

func CodeMatches(key string, sessionID int64, now time.Time, offered string) bool {
	offered = strings.TrimSpace(offered)
	if len(offered) != CodeDigits {
		return false
	}
	slice := Slice(now, CodeSliceSeconds)
	return sameString(Code(key, sessionID, slice), offered) ||
		sameString(Code(key, sessionID, slice-1), offered)
}

func sameString(want, got string) bool {
	return hmac.Equal([]byte(want), []byte(got))
}

const earthRadiusM = 6371008.8

func DistanceM(fromLat, fromLng, toLat, toLng float64) float64 {
	phi1 := fromLat * math.Pi / 180
	phi2 := toLat * math.Pi / 180
	deltaPhi := phi2 - phi1
	deltaLambda := (toLng - fromLng) * math.Pi / 180
	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	return 2 * earthRadiusM * math.Asin(math.Min(1, math.Sqrt(a)))
}

type Position struct {
	Lat       float64
	Lng       float64
	AccuracyM float64
	IsMock    bool
}

type Anchor struct {
	Lat       float64
	Lng       float64
	UpdatedAt time.Time
	Known     bool
}

func CheckPosition(student Position, teacher Anchor, now time.Time) (float64, string) {
	if student.IsMock {
		return 0, MessageMockLocation
	}
	if student.Lat == 0 && student.Lng == 0 {
		return 0, MessageNoPosition
	}
	if student.AccuracyM <= 0 || student.AccuracyM > MaxAccuracyM {
		return 0, MessageWeakSignal
	}
	if !teacher.Known || teacher.UpdatedAt.IsZero() ||
		now.Sub(teacher.UpdatedAt) > TeacherFreshness {
		return 0, MessageTeacherStale
	}
	distance := DistanceM(teacher.Lat, teacher.Lng, student.Lat, student.Lng)
	if distance > MaxDistanceM {
		return distance, MessageTooFar
	}
	return distance, ""
}

func ValidDuration(minutes int) bool {
	window := time.Duration(minutes) * time.Minute
	return window >= MinDuration && window <= MaxDuration
}

func Percentage(present, held int) float64 {
	if held <= 0 {
		return 0
	}
	return float64(present) * 100 / float64(held)
}

func BelowThreshold(present, held int) bool {
	return held >= AlertMinSessions && Percentage(present, held) < AlertPercentage
}

func AlertDue(present, held int, lastSent, now time.Time) bool {
	if !BelowThreshold(present, held) {
		return false
	}
	if lastSent.IsZero() {
		return true
	}
	return now.Sub(lastSent) >= AlertCooldown
}

func NoteRequired(approvedChanges int) bool {
	return approvedChanges >= SemesterChangeLimit
}

func CoversSection(sessionSection, studentSection string) bool {
	sessionSection = strings.ToUpper(strings.TrimSpace(sessionSection))
	studentSection = strings.ToUpper(strings.TrimSpace(studentSection))
	return sessionSection == SectionAll || sessionSection == studentSection
}
