package handler

import (
	"github.com/raHI-79/250-project/backend/internal/config"
	"net/http"
	"strings"
)

const (
	countryHeader = "CF-IPCountry"
	regionHeader  = "X-Vercel-IP-Country"
	maxDeviceName = 80
)

type Visitors struct {
	salt string
}

func NewVisitors(cfg *config.Config) *Visitors {
	salt := ""
	if cfg != nil {
		salt = cfg.AnalyticsSalt
	}
	return &Visitors{salt: salt}
}

func (v *Visitors) Address(r *http.Request) (string, string) {
	return HashAddress(v.salt, RequestAddress(r)), RequestCountry(r)
}

func RequestCountry(r *http.Request) string {
	for _, header := range []string{countryHeader, regionHeader} {
		value := strings.ToUpper(strings.TrimSpace(r.Header.Get(header)))
		if value != "" && value != "XX" && value != "T1" {
			return value
		}
	}
	return ""
}

func RequestDevice(r *http.Request) string {
	agent := strings.TrimSpace(r.UserAgent())
	if agent == "" {
		return "Unknown device"
	}
	name := deviceName(agent)
	if len(name) > maxDeviceName {
		return name[:maxDeviceName]
	}
	return name
}

func deviceName(agent string) string {
	lowered := strings.ToLower(agent)
	platform := "Computer"
	switch {
	case strings.Contains(lowered, "android"):
		platform = "Android"
	case strings.Contains(lowered, "iphone"), strings.Contains(lowered, "ipad"):
		platform = "iPhone or iPad"
	case strings.Contains(lowered, "windows"):
		platform = "Windows"
	case strings.Contains(lowered, "mac os"), strings.Contains(lowered, "macintosh"):
		platform = "Mac"
	case strings.Contains(lowered, "linux"):
		platform = "Linux"
	}
	browser := ""
	switch {
	case strings.Contains(lowered, "edg/"):
		browser = "Edge"
	case strings.Contains(lowered, "opr/"), strings.Contains(lowered, "opera"):
		browser = "Opera"
	case strings.Contains(lowered, "firefox"):
		browser = "Firefox"
	case strings.Contains(lowered, "chrome"):
		browser = "Chrome"
	case strings.Contains(lowered, "safari"):
		browser = "Safari"
	}
	if browser == "" {
		return platform
	}
	return platform + " · " + browser
}
