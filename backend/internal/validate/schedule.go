package validate

import (
	"fmt"
	"strings"
)

const (
	ItemClass        = "class"
	ItemLab          = "lab"
	ItemExam         = "exam"
	ModeRoom         = "room"
	ModeOnline       = "online"
	MaxRoomLength    = 40
	MaxMessageLength = 500
	MaxSyllabus      = 2000
)

var (
	itemKinds = []string{ItemClass, ItemLab, ItemExam}
	itemModes = []string{ModeRoom, ModeOnline}
)

func ItemKind(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	return oneOf(label, strings.ToLower(trimmed), itemKinds)
}

func ClassOrExam(label, value string) (string, error) {
	kind, err := ItemKind(label, value)
	if err != nil {
		return "", err
	}
	if kind == ItemLab {
		return "", fmt.Errorf("%s must be %s or %s.", label, ItemClass, ItemExam)
	}
	return kind, nil
}

func ItemMode(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	return oneOf(label, strings.ToLower(trimmed), itemModes)
}

func Room(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	if len(trimmed) > MaxRoomLength {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxRoomLength)
	}
	return trimmed, nil
}

func ShortText(label, value string, limit int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len([]rune(trimmed)) > limit {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, limit)
	}
	return trimmed, nil
}

func Message(label, value string) (string, error) {
	trimmed, err := ShortText(label, value, MaxMessageLength)
	if err != nil {
		return "", err
	}
	if trimmed == "" {
		return "", fmt.Errorf("%s is required.", label)
	}
	return trimmed, nil
}

func Slot(mode, room, link string) (string, string, error) {
	if mode == ModeOnline {
		return "", link, nil
	}
	cleaned, err := Room("Room", room)
	if err != nil {
		return "", "", err
	}
	return cleaned, link, nil
}
