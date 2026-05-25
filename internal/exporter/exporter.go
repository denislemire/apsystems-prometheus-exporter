package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/denislemire/apsystems-prometheus-exporter/internal/api"
	"github.com/denislemire/apsystems-prometheus-exporter/internal/config"
	"github.com/denislemire/apsystems-prometheus-exporter/internal/layout"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type batchEnergyData struct {
	Energy []string `json:"energy"`
}

type batchPowerData struct {
	Time  []string             `json:"time"`
	Power map[string][]string  `json:"power"`
}

type summaryData struct {
	Today string `json:"today"`
	Month string `json:"month"`
}

type detailsData struct {
	Light int `json:"light"`
}

type Exporter struct {
	cfg    config.Config
	client *api.Client
	layout layout.File
	loc    *time.Location

	panelEnergy *prometheus.GaugeVec
	panelPower  *prometheus.GaugeVec
	systemToday *prometheus.GaugeVec
	systemMonth *prometheus.GaugeVec
	systemStatus *prometheus.GaugeVec
	scrapeOK    prometheus.Gauge
	apiCalls    prometheus.Counter
	lastScrape  prometheus.Gauge
	scrapeCount int
}

func New(cfg config.Config, panels layout.File) (*Exporter, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	e := &Exporter{
		cfg:    cfg,
		layout: panels,
		loc:    loc,
		panelEnergy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apsystems_panel_energy_today_kwh",
			Help: "Panel cumulative energy today in kWh (inverter-reported)",
		}, []string{"uid", "channel", "array", "row", "col"}),
		panelPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apsystems_panel_power_watts",
			Help: "Panel instantaneous power in watts (latest cloud sample)",
		}, []string{"uid", "channel", "array", "row", "col"}),
		systemToday: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apsystems_system_energy_today_kwh",
			Help: "Whole-system cumulative energy today in kWh",
		}, []string{"sid"}),
		systemMonth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apsystems_system_energy_month_kwh",
			Help: "Whole-system cumulative energy this month in kWh",
		}, []string{"sid"}),
		systemStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apsystems_system_status",
			Help: "ECU status light: 1=green 2=yellow 3=red 4=grey",
		}, []string{"sid"}),
		scrapeOK: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "apsystems_exporter_scrape_success",
			Help: "1 on successful scrape",
		}),
		apiCalls: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "apsystems_exporter_api_calls_total",
			Help: "APSystems OpenAPI calls made",
		}),
		lastScrape: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "apsystems_exporter_last_scrape_timestamp_seconds",
			Help: "Unix timestamp of last successful scrape",
		}),
	}
	e.client = api.New(cfg.APIBase, cfg.AppID, cfg.AppSecret, e.apiCalls.Inc)
	prometheus.MustRegister(
		e.panelEnergy, e.panelPower, e.systemToday, e.systemMonth, e.systemStatus,
		e.scrapeOK, e.apiCalls, e.lastScrape,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "apsystems_exporter_info",
			Help: "Exporter build info",
			ConstLabels: prometheus.Labels{
				"version": Version,
			},
		}, func() float64 { return 1 }),
	)
	return e, nil
}

func (e *Exporter) Handler() http.Handler {
	return promhttp.Handler()
}

func (e *Exporter) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", e.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	server := &http.Server{Addr: e.cfg.ListenAddr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "err", err)
		}
	}()

	slog.Info("exporter started",
		"listen", e.cfg.ListenAddr,
		"sid", e.cfg.SystemID,
		"ecu", e.cfg.ECUID,
		"interval", e.cfg.ScrapeInterval,
		"timezone", e.cfg.Timezone,
		"panels", len(e.layout.Panels),
	)

	if err := e.Scrape(ctx); err != nil {
		slog.Warn("initial scrape failed", "err", err)
	}

	ticker := time.NewTicker(e.cfg.ScrapeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := e.Scrape(ctx); err != nil {
				slog.Error("scrape failed", "err", err)
				e.scrapeOK.Set(0)
			}
		}
	}
}

func (e *Exporter) Scrape(_ context.Context) error {
	if !e.inSolarWindow() {
		slog.Debug("outside solar window, skipping API calls",
			"start", e.cfg.SolarStartHour, "end", e.cfg.SolarEndHour)
		e.scrapeOK.Set(1)
		return nil
	}

	e.scrapeCount++
	today := time.Now().In(e.loc).Format("2006-01-02")
	batchPath := fmt.Sprintf("/user/api/v2/systems/%s/devices/inverter/batch/energy/%s", e.cfg.SystemID, e.cfg.ECUID)
	params := url.Values{"energy_level": {"energy"}, "date_range": {today}}

	energyResp, err := e.client.Get(batchPath, params)
	if err != nil {
		return fmt.Errorf("batch energy: %w", err)
	}
	var energy batchEnergyData
	if err := json.Unmarshal(energyResp.Data, &energy); err != nil {
		return fmt.Errorf("decode batch energy: %w", err)
	}
	e.setBatchEnergy(energy)

	params.Set("energy_level", "power")
	powerResp, err := e.client.Get(batchPath, params)
	if err != nil {
		return fmt.Errorf("batch power: %w", err)
	}
	var power batchPowerData
	if err := json.Unmarshal(powerResp.Data, &power); err != nil {
		return fmt.Errorf("decode batch power: %w", err)
	}
	e.setBatchPower(power)

	if e.scrapeCount%e.cfg.SummaryEveryN == 0 {
		if err := e.fetchSummary(); err != nil {
			return err
		}
	}

	e.lastScrape.Set(float64(time.Now().Unix()))
	e.scrapeOK.Set(1)
	slog.Info("scrape ok", "panels", len(energy.Energy), "count", e.scrapeCount)
	return nil
}

func (e *Exporter) fetchSummary() error {
	summaryPath := fmt.Sprintf("/user/api/v2/systems/summary/%s", e.cfg.SystemID)
	summaryResp, err := e.client.Get(summaryPath, nil)
	if err != nil {
		return fmt.Errorf("system summary: %w", err)
	}
	var summary summaryData
	if err := json.Unmarshal(summaryResp.Data, &summary); err != nil {
		return fmt.Errorf("decode summary: %w", err)
	}
	e.systemToday.WithLabelValues(e.cfg.SystemID).Set(parseFloat(summary.Today))
	e.systemMonth.WithLabelValues(e.cfg.SystemID).Set(parseFloat(summary.Month))

	detailsPath := fmt.Sprintf("/user/api/v2/systems/details/%s", e.cfg.SystemID)
	detailsResp, err := e.client.Get(detailsPath, nil)
	if err != nil {
		return fmt.Errorf("system details: %w", err)
	}
	var details detailsData
	if err := json.Unmarshal(detailsResp.Data, &details); err != nil {
		return fmt.Errorf("decode details: %w", err)
	}
	e.systemStatus.WithLabelValues(e.cfg.SystemID).Set(float64(details.Light))
	slog.Info("summary fetched", "today_kwh", summary.Today, "status", details.Light)
	return nil
}

func (e *Exporter) setBatchEnergy(data batchEnergyData) {
	for _, item := range data.Energy {
		uid, channel, kwh, ok := parseBatchItem(item)
		if !ok {
			continue
		}
		l := e.layout.Labels(uid + "-" + channel)
		e.panelEnergy.WithLabelValues(l.UID, l.Channel, l.Array, l.Row, l.Col).Set(kwh)
	}
}

func (e *Exporter) setBatchPower(data batchPowerData) {
	if len(data.Time) == 0 || len(data.Power) == 0 {
		return
	}
	idx := len(data.Time) - 1
	for key, series := range data.Power {
		if idx >= len(series) {
			continue
		}
		w := parseFloat(series[idx])
		l := e.layout.Labels(key)
		e.panelPower.WithLabelValues(l.UID, l.Channel, l.Array, l.Row, l.Col).Set(w)
	}
}

func (e *Exporter) inSolarWindow() bool {
	hour := time.Now().In(e.loc).Hour()
	return hour >= e.cfg.SolarStartHour && hour < e.cfg.SolarEndHour
}

func parseBatchItem(item string) (uid, channel string, kwh float64, ok bool) {
	parts := strings.Split(item, "-")
	if len(parts) < 3 {
		return "", "", 0, false
	}
	kwhStr := parts[len(parts)-1]
	channel = parts[len(parts)-2]
	uid = strings.Join(parts[:len(parts)-2], "-")
	v, err := strconv.ParseFloat(kwhStr, 64)
	if err != nil {
		return "", "", 0, false
	}
	return uid, channel, v, true
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
