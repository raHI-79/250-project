package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

const (
	defaultPort           = "8080"
	defaultAppEnv         = "development"
	defaultTimezone       = "Asia/Dhaka"
	defaultJWTTTL         = "24h"
	defaultFrontendOrigin = "http://localhost:5173"
	DecisionStateBefore   = "before"
	DecisionStateOpen     = "open"
	DecisionStateClosed   = "closed"
	defaultDecisionStart  = "16:00"
	defaultDecisionEnd    = "23:00"
	defaultDaysAhead      = 7
	defaultUploadDir      = "uploads"
	defaultMaxUploadMB    = 10
	maxDaysAhead          = 31
	defaultSMTPPort       = 587
	defaultStudentDomain  = "student.sust.edu"
	minJWTSecretLength    = 32
	defaultMailFromName   = "SUST CSE Portal"
	defaultMaxConns       = 10
	maxDatabaseConns      = 100
	FileStorageDisk       = "disk"
	FileStorageDatabase   = "database"
	MailProviderSMTP      = "smtp"
	MailProviderBrevo     = "brevo"
)

var (
	ErrMissingDatabaseURL = errors.New("config: DATABASE_URL is required (copy backend/.env.example to backend/.env and set your database password)")
	ErrMissingJWTSecret   = errors.New("config: JWT_SECRET is required (set a long random value in backend/.env)")
	ErrShortJWTSecret     = fmt.Errorf("config: JWT_SECRET must be at least %d characters long", minJWTSecretLength)
)

type Config struct {
	Port            string
	AppEnv          string
	Timezone        string
	Location        *time.Location
	DatabaseURL     string
	TestDatabaseURL string
	JWTSecret       string
	JWTTTL          time.Duration
	FrontendOrigin  string
	DecisionStart   string
	DecisionEnd     string
	DecisionFrom    int
	DecisionTo      int
	DaysAhead       int
	UploadDir       string
	MaxUploadMB     int
	FileStorage     string
	StaticDir       string
	MailProvider    string
	MailFromName    string
	BrevoAPIKey     string
	DBMaxConns      int
	AdminIdentifier string
	AdminPassword   string
	AdminEmail      string
	AdminPhone      string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	MailFrom        string
	StudentDomain   string
	GoogleClientID  string
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
}

func (c *Config) PushEnabled() bool {
	return c.VAPIDPublicKey != "" && c.VAPIDPrivateKey != "" && c.VAPIDSubject != ""
}

func Load() (*Config, error) {
	if err := LoadEnvFile(envFileName); err != nil {
		return nil, err
	}
	cfg := &Config{
		Port:            getEnv("PORT", defaultPort),
		AppEnv:          getEnv("APP_ENV", defaultAppEnv),
		Timezone:        getEnv("APP_TIMEZONE", defaultTimezone),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		TestDatabaseURL: os.Getenv("TEST_DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		FrontendOrigin:  getEnv("FRONTEND_ORIGIN", defaultFrontendOrigin),
		UploadDir:       getEnv("UPLOAD_DIR", defaultUploadDir),
		FileStorage:     strings.ToLower(getEnv("FILE_STORAGE", FileStorageDisk)),
		StaticDir:       strings.TrimSpace(os.Getenv("STATIC_DIR")),
		MailProvider:    strings.ToLower(getEnv("MAIL_PROVIDER", MailProviderSMTP)),
		MailFromName:    getEnv("MAIL_FROM_NAME", defaultMailFromName),
		BrevoAPIKey:     strings.TrimSpace(os.Getenv("BREVO_API_KEY")),
		AdminIdentifier: os.Getenv("ADMIN_IDENTIFIER"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminPhone:      os.Getenv("ADMIN_PHONE"),
		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		MailFrom:        os.Getenv("MAIL_FROM"),
		StudentDomain:   strings.ToLower(getEnv("STUDENT_EMAIL_DOMAIN", defaultStudentDomain)),
		GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
		VAPIDPublicKey:  strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY")),
		VAPIDPrivateKey: strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY")),
		VAPIDSubject:    strings.TrimSpace(os.Getenv("VAPID_SUBJECT")),
	}
	if cfg.DatabaseURL == "" {
		return nil, ErrMissingDatabaseURL
	}
	if cfg.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	if len(cfg.JWTSecret) < minJWTSecretLength {
		return nil, ErrShortJWTSecret
	}
	rawTTL := getEnv("JWT_TTL", defaultJWTTTL)
	ttl, err := time.ParseDuration(rawTTL)
	if err != nil {
		return nil, fmt.Errorf("config: invalid JWT_TTL %q: %w", rawTTL, err)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("config: JWT_TTL must be a positive duration, got %q", rawTTL)
	}
	cfg.JWTTTL = ttl
	cfg.DecisionStart = getEnv("DECISION_WINDOW_START", defaultDecisionStart)
	cfg.DecisionFrom, err = clockMinutes("DECISION_WINDOW_START", cfg.DecisionStart)
	if err != nil {
		return nil, err
	}
	cfg.DecisionEnd = getEnv("DECISION_WINDOW_END", defaultDecisionEnd)
	cfg.DecisionTo, err = clockMinutes("DECISION_WINDOW_END", cfg.DecisionEnd)
	if err != nil {
		return nil, err
	}
	if cfg.DecisionFrom >= cfg.DecisionTo {
		return nil, fmt.Errorf("config: DECISION_WINDOW_START %q must be earlier than DECISION_WINDOW_END %q",
			cfg.DecisionStart, cfg.DecisionEnd)
	}
	rawDaysAhead := getEnv("SCHEDULE_DAYS_AHEAD", strconv.Itoa(defaultDaysAhead))
	daysAhead, err := strconv.Atoi(rawDaysAhead)
	if err != nil {
		return nil, fmt.Errorf("config: invalid SCHEDULE_DAYS_AHEAD %q: %w", rawDaysAhead, err)
	}
	if daysAhead < 1 || daysAhead > maxDaysAhead {
		return nil, fmt.Errorf("config: SCHEDULE_DAYS_AHEAD must be a number between 1 and %d, got %q",
			maxDaysAhead, rawDaysAhead)
	}
	cfg.DaysAhead = daysAhead
	if cfg.FileStorage != FileStorageDisk && cfg.FileStorage != FileStorageDatabase {
		return nil, fmt.Errorf("config: FILE_STORAGE must be %s or %s, got %q",
			FileStorageDisk, FileStorageDatabase, cfg.FileStorage)
	}
	if cfg.MailProvider != MailProviderSMTP && cfg.MailProvider != MailProviderBrevo {
		return nil, fmt.Errorf("config: MAIL_PROVIDER must be %s or %s, got %q",
			MailProviderSMTP, MailProviderBrevo, cfg.MailProvider)
	}
	rawMaxConns := getEnv("DB_MAX_CONNS", strconv.Itoa(defaultMaxConns))
	maxConns, err := strconv.Atoi(rawMaxConns)
	if err != nil || maxConns < 1 || maxConns > maxDatabaseConns {
		return nil, fmt.Errorf("config: DB_MAX_CONNS must be a number between 1 and %d, got %q",
			maxDatabaseConns, rawMaxConns)
	}
	cfg.DBMaxConns = maxConns
	rawMaxUpload := getEnv("MAX_UPLOAD_MB", strconv.Itoa(defaultMaxUploadMB))
	maxUpload, err := strconv.Atoi(rawMaxUpload)
	if err != nil {
		return nil, fmt.Errorf("config: invalid MAX_UPLOAD_MB %q: %w", rawMaxUpload, err)
	}
	if maxUpload <= 0 {
		return nil, fmt.Errorf("config: MAX_UPLOAD_MB must be a positive number of megabytes, got %q", rawMaxUpload)
	}
	cfg.MaxUploadMB = maxUpload
	rawSMTPPort := getEnv("SMTP_PORT", strconv.Itoa(defaultSMTPPort))
	smtpPort, err := strconv.Atoi(rawSMTPPort)
	if err != nil {
		return nil, fmt.Errorf("config: invalid SMTP_PORT %q: %w", rawSMTPPort, err)
	}
	if smtpPort <= 0 || smtpPort > 65535 {
		return nil, fmt.Errorf("config: SMTP_PORT must be a port number between 1 and 65535, got %q", rawSMTPPort)
	}
	cfg.SMTPPort = smtpPort
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("config: invalid APP_TIMEZONE %q: %w", cfg.Timezone, err)
	}
	cfg.Location = loc
	return cfg, nil
}

func (c *Config) FilesInDatabase() bool {
	return c.FileStorage == FileStorageDatabase
}

func (c *Config) ServesStatic() bool {
	return c.StaticDir != ""
}

func (c *Config) Now() time.Time {
	return time.Now().In(c.Location)
}

func (c *Config) DecisionOpen(now time.Time) bool {
	minutes := now.Hour()*60 + now.Minute()
	return minutes >= c.DecisionFrom && minutes < c.DecisionTo
}

func (c *Config) DecisionClosed(now time.Time) bool {
	return now.Hour()*60+now.Minute() >= c.DecisionTo
}

func (c *Config) DecisionState(now time.Time) string {
	switch {
	case c.DecisionOpen(now):
		return DecisionStateOpen
	case c.DecisionClosed(now):
		return DecisionStateClosed
	default:
		return DecisionStateBefore
	}
}

func (c *Config) DecisionMinutesLeft(now time.Time) int {
	if !c.DecisionOpen(now) {
		return 0
	}
	return c.DecisionTo - (now.Hour()*60 + now.Minute())
}

func clockMinutes(key, value string) (int, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("config: %s must look like 17:00, got %q", key, value)
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func (c *Config) HasAdminSeed() bool {
	return c.AdminIdentifier != "" && c.AdminPassword != ""
}

func (c *Config) HasMailer() bool {
	return c.SMTPHost != ""
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func (c *Config) GoogleEnabled() bool {
	return strings.TrimSpace(c.GoogleClientID) != ""
}

func (c *Config) StudentEmailSuffix() string {
	return "@" + c.StudentDomain
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
