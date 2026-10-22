package validate

import (
	"fmt"
	"strings"
)

const (
	MaxMessageBody   = 4000
	MaxNote          = 500
	ScopeBatch       = "batch"
	ScopeSection     = "section"
	ScopeAllStudents = "all_students"
	ScopeAllTeachers = "all_teachers"
	NoteTeacher      = "teacher"
	NoteStudents     = "students"
	NoteBoth         = "both"
	SourceAll        = "all"
	SourceAdmin      = "admin"
	SourceTeacher    = "teacher"
	SourceCR         = "cr"
	RoleStudent      = "student"
	RoleCR           = "cr"
	RoleTeacher      = "teacher"
	RoleAdmin        = "admin"
)

var (
	conversationScopes = []string{ScopeBatch, ScopeSection, ScopeAllStudents, ScopeAllTeachers}
	noteAudiences      = []string{NoteTeacher, NoteStudents, NoteBoth}
	noticeSources      = []string{SourceAll, SourceAdmin, SourceTeacher, SourceCR}
	messageRoles       = []string{RoleStudent, RoleCR, RoleTeacher, RoleAdmin}
)

func CanStartDirect(viewerRole, targetRole string) bool {
	return !(viewerRole == RoleStudent && targetRole == RoleStudent)
}

func ListsPlainStudents(viewerRole string) bool {
	return viewerRole != RoleStudent
}

func MessageBody(label, value string, hasAttachment bool) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" && !hasAttachment {
		return "", fmt.Errorf("%s is required when nothing is attached.", label)
	}
	if len(trimmed) > MaxMessageBody {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxMessageBody)
	}
	return trimmed, nil
}

func ConversationScope(label, value string) (string, error) {
	return oneOf(label, value, conversationScopes)
}

func NoteAudience(label, value string) (string, error) {
	return oneOf(label, value, noteAudiences)
}

func NoticeSource(label, value string) (string, error) {
	return oneOf(label, value, noticeSources)
}

func MessageRole(label, value string) (string, error) {
	return oneOf(label, value, messageRoles)
}

func Note(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > MaxNote {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxNote)
	}
	return trimmed, nil
}

func OptionalLink(label, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return Link(label, value)
}
