package validate

import (
	"fmt"
	"strings"
)

const (
	MaxQuestionLength = 300
	MinPollOptions    = 2
	MaxPollOptions    = 10
	PollScopeSection  = "section"
	PollScopeBatch    = "batch"
	PollScopeAll      = "all_batches"
)

var pollScopes = []string{PollScopeSection, PollScopeBatch, PollScopeAll}

func PollScope(label, value string) (string, error) {
	return oneOf(label, value, pollScopes)
}

func Question(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required.", label)
	}
	if len(trimmed) > MaxQuestionLength {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxQuestionLength)
	}
	return trimmed, nil
}

func PollOptions(label string, values []string) ([]string, error) {
	options := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("%s must not contain an empty option.", label)
		}
		if len(trimmed) > MaxFieldLength {
			return nil, fmt.Errorf("%s must be at most %d characters long each.", label, MaxFieldLength)
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			return nil, fmt.Errorf("%s must not repeat the same option.", label)
		}
		seen[key] = true
		options = append(options, trimmed)
	}
	if len(options) < MinPollOptions || len(options) > MaxPollOptions {
		return nil, fmt.Errorf("%s must hold between %d and %d options.", label, MinPollOptions, MaxPollOptions)
	}
	return options, nil
}

func PollChoices(label string, value, options int) (int, error) {
	if value == 0 {
		return 1, nil
	}
	if value < 1 || value > options {
		return 0, fmt.Errorf("%s must be a number between 1 and %d.", label, options)
	}
	return value, nil
}
