package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	DefaultAPIBase        = "https://api.apsystemsema.com:9282"
	DefaultListenAddr     = ":9921"
	DefaultScrapeInterval = 2 * time.Hour
	DefaultTimezone       = "UTC"
	DefaultSolarStartHour = 6
	DefaultSolarEndHour   = 22
	DefaultSummaryEveryN  = 6
)

type Config struct {
	APIBase        string
	AppID          string
	AppSecret      string
	SystemID       string
	ECUID          string
	ListenAddr     string
	ScrapeInterval time.Duration
	Timezone       string
	SolarStartHour int
	SolarEndHour   int
	SummaryEveryN  int
	LayoutPath     string
}

func Load() (Config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	apiBase := fs.String("api-base", envOr("APS_API_BASE", DefaultAPIBase), "APSystems OpenAPI base URL")
	appID := fs.String("app-id", os.Getenv("APS_APP_ID"), "OpenAPI App ID")
	appSecret := fs.String("app-secret", os.Getenv("APS_APP_SECRET"), "OpenAPI App Secret")
	systemID := fs.String("sid", os.Getenv("APS_SID"), "System ID")
	ecuID := fs.String("ecu-id", os.Getenv("APS_ECU_ID"), "ECU ID")
	listenAddr := fs.String("listen", envOr("EXPORTER_LISTEN", DefaultListenAddr), "HTTP listen address")
	scrapeInterval := fs.Duration("scrape-interval", durationEnv("SCRAPE_INTERVAL", DefaultScrapeInterval), "Poll interval")
	timezone := fs.String("timezone", envOr("TZ", DefaultTimezone), "IANA timezone for date/window")
	solarStart := fs.Int("solar-start-hour", intEnv("SOLAR_START_HOUR", DefaultSolarStartHour), "Start hour (local) for API polling")
	solarEnd := fs.Int("solar-end-hour", intEnv("SOLAR_END_HOUR", DefaultSolarEndHour), "End hour (local) for API polling")
	summaryEveryN := fs.Int("summary-every-n", intEnv("SUMMARY_EVERY_N", DefaultSummaryEveryN), "Fetch system summary every N scrapes")
	layoutPath := fs.String("panels-layout", envOr("APS_PANELS_LAYOUT", "/etc/apsystems/panels-layout.json"), "Panel layout JSON path")

	_ = fs.Parse(os.Args[1:])

	cfg := Config{
		APIBase:        *apiBase,
		AppID:          *appID,
		AppSecret:      *appSecret,
		SystemID:       *systemID,
		ECUID:          *ecuID,
		ListenAddr:     *listenAddr,
		ScrapeInterval: *scrapeInterval,
		Timezone:       *timezone,
		SolarStartHour: *solarStart,
		SolarEndHour:   *solarEnd,
		SummaryEveryN:  *summaryEveryN,
		LayoutPath:     *layoutPath,
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch {
	case c.AppID == "":
		return errors.New("APS_APP_ID (or --app-id) is required")
	case c.AppSecret == "":
		return errors.New("APS_APP_SECRET (or --app-secret) is required")
	case c.SystemID == "":
		return errors.New("APS_SID (or --sid) is required")
	case c.ECUID == "":
		return errors.New("APS_ECU_ID (or --ecu-id) is required")
	case c.SolarStartHour < 0 || c.SolarStartHour > 23:
		return fmt.Errorf("solar start hour out of range: %d", c.SolarStartHour)
	case c.SolarEndHour < 1 || c.SolarEndHour > 24:
		return fmt.Errorf("solar end hour out of range: %d", c.SolarEndHour)
	case c.SummaryEveryN < 1:
		return fmt.Errorf("summary-every-n must be >= 1")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
