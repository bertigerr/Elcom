package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath     string
	RawMailDir string
	OutputDir  string

	ElcomAPIBaseURL        string
	ElcomAPIToken          string
	ElcomRateLimitRPS      int
	ElcomTimeoutMs         int
	IncrementalLookbackHrs int
	IncrementalLookbackDay int

	MatchOKThreshold     float64
	MatchReviewThreshold float64
	MatchGapThreshold    float64

	GmailClientID     string
	GmailClientSecret string
	GmailRedirectURI  string
	GmailRefreshToken string

	IMAPHost     string
	IMAPPort     int
	IMAPSecure   bool
	IMAPUser     string
	IMAPPassword string
	IMAPMarkSeen bool

	MailListenerProvider     string
	MailListenerLabel        string
	MailListenerIntervalSec  int
	MailListenerFetchMax     int
	MailListenerProcessBatch int
	MailListenerAutoExport   bool
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		DBPath:     getEnv("DB_PATH", filepath.Join(cwd, "data", "app.db")),
		RawMailDir: getEnv("MAIL_RAW_DIR", filepath.Join(cwd, "data", "raw")),
		OutputDir:  getEnv("OUTPUT_DIR", filepath.Join(cwd, "out")),

		ElcomAPIBaseURL:        getEnv("ELCOM_API_BASE_URL", "https://online.el-com.ru/api/v1"),
		ElcomAPIToken:          getEnv("ELCOM_API_TOKEN", ""),
		ElcomRateLimitRPS:      getEnvInt("ELCOM_RATE_LIMIT_RPS", 5),
		ElcomTimeoutMs:         getEnvInt("ELCOM_TIMEOUT_MS", 30000),
		IncrementalLookbackHrs: getEnvInt("ELCOM_INCREMENTAL_HOURS", 24),
		IncrementalLookbackDay: getEnvInt("ELCOM_INCREMENTAL_DAYS", 2),

		MatchOKThreshold:     getEnvFloat("MATCH_OK_THRESHOLD", 0.90),
		MatchReviewThreshold: getEnvFloat("MATCH_REVIEW_THRESHOLD", 0.72),
		MatchGapThreshold:    getEnvFloat("MATCH_GAP_THRESHOLD", 0.08),

		GmailClientID:     getEnv("GMAIL_CLIENT_ID", ""),
		GmailClientSecret: getEnv("GMAIL_CLIENT_SECRET", ""),
		GmailRedirectURI:  getEnv("GMAIL_REDIRECT_URI", "https://developers.google.com/oauthplayground"),
		GmailRefreshToken: getEnv("GMAIL_REFRESH_TOKEN", ""),

		IMAPHost:     getEnv("IMAP_HOST", ""),
		IMAPPort:     getEnvInt("IMAP_PORT", 993),
		IMAPSecure:   getEnvBool("IMAP_SECURE", true),
		IMAPUser:     getEnv("IMAP_USER", ""),
		IMAPPassword: getEnv("IMAP_PASSWORD", ""),
		IMAPMarkSeen: getEnvBool("IMAP_MARK_SEEN", false),

		MailListenerProvider:     getEnv("MAIL_LISTENER_PROVIDER", "gmail"),
		MailListenerLabel:        getEnv("MAIL_LISTENER_LABEL", "INBOX"),
		MailListenerIntervalSec:  getEnvInt("MAIL_LISTENER_INTERVAL_SEC", 30),
		MailListenerFetchMax:     getEnvInt("MAIL_LISTENER_FETCH_MAX", 20),
		MailListenerProcessBatch: getEnvInt("MAIL_LISTENER_PROCESS_BATCH", 20),
		MailListenerAutoExport:   getEnvBool("MAIL_LISTENER_AUTO_EXPORT", true),
	}

	return cfg, nil
}

func (c Config) Require(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("missing required env var: %s", name)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(getEnv(key, "")))
	if value == "" {
		return fallback
	}
	if value == "1" || value == "true" || value == "yes" || value == "on" {
		return true
	}
	if value == "0" || value == "false" || value == "no" || value == "off" {
		return false
	}
	return fallback
}
