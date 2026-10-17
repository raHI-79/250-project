package validate

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	BatchAll          = "ALL"
	MaxDescription    = 2000
	MaxLinkLength     = 500
	MaxSearchLength   = 120
	VisibilityMine    = "mine"
	VisibilityBatch   = "batch"
	VisibilityAll     = "all"
	NoticeTargetOwn   = "own"
	NoticeScopeMine   = "my"
	NoticeScopeOthers = "other"
)

var (
	visibilities = []string{VisibilityMine, VisibilityBatch, VisibilityAll}
	noticeScopes = []string{NoticeScopeMine, NoticeScopeOthers}
)

func Description(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > MaxDescription {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxDescription)
	}
	return trimmed, nil
}

func Search(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > MaxSearchLength {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxSearchLength)
	}
	return trimmed, nil
}

func Link(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required.", label)
	}
	if len(trimmed) > MaxLinkLength {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxLinkLength)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("%s must be a full link that starts with http:// or https://.", label)
	}
	return trimmed, nil
}

func BatchOrAll(label, value string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(trimmed, BatchAll) {
		return BatchAll, nil
	}
	return Batch(label, trimmed)
}

func Visibility(label, value string) (string, error) {
	return oneOf(label, value, visibilities)
}

func NoticeScope(label, value string) (string, error) {
	return oneOf(label, value, noticeScopes)
}

func oneOf(label, value string, allowed []string) (string, error) {
	trimmed, err := Required(label, value)
	if err != nil {
		return "", err
	}
	trimmed = strings.ToLower(trimmed)
	for _, option := range allowed {
		if trimmed == option {
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("%s must be one of %s.", label, strings.Join(allowed, ", "))
}
