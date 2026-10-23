package config

import (
	"fmt"
	"net/url"
	"strconv"
)

// MaxAmount is the supported whole-rupiah technical amount cap, not a credit policy.
const MaxAmount int64 = 1_000_000_000_000

// Config contains validated process settings.
type Config struct {
	DatabaseURL     string
	Origin          string
	Address         string
	SecureCookies   bool
	ApprovalLimit   int64
	InterestEnabled bool
}

// Load parses environment settings without disclosing secret values.
func Load(getenv func(string) string) (Config, error) {
	value := func(key, fallback string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fallback
	}
	c := Config{DatabaseURL: value("DATABASE_URL", "postgres:///credit_union?host=/var/run/postgresql"),
		Origin: value("APP_ORIGIN", "http://localhost:3000"), Address: value("API_ADDR", "127.0.0.1:8080")}
	u, err := url.Parse(c.Origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return Config{}, fmt.Errorf("APP_ORIGIN must be an http(s) origin without a path")
	}
	c.SecureCookies, err = strconv.ParseBool(value("COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE must be true or false")
	}
	c.InterestEnabled, err = strconv.ParseBool(value("SAVINGS_INTEREST_ENABLED", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("SAVINGS_INTEREST_ENABLED must be true or false")
	}
	c.ApprovalLimit, err = strconv.ParseInt(value("APPROVAL_LIMIT_IDR", "0"), 10, 64)
	if err != nil || c.ApprovalLimit < 0 || c.ApprovalLimit > MaxAmount {
		return Config{}, fmt.Errorf("invalid APPROVAL_LIMIT_IDR")
	}
	return c, nil
}
