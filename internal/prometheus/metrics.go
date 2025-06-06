// Package prometheus provides Prometheus metrics for Mailpit
package prometheus

import (
	"net/http"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/stats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Registry is the Prometheus registry for Mailpit metrics
	Registry = prometheus.NewRegistry()

	// Metrics
	totalMessages    prometheus.Gauge
	unreadMessages   prometheus.Gauge
	databaseSize     prometheus.Gauge
	messagesDeleted  prometheus.Counter
	smtpAccepted     prometheus.Counter
	smtpRejected     prometheus.Counter
	smtpIgnored      prometheus.Counter
	smtpAcceptedSize prometheus.Counter
	uptime           prometheus.Gauge
	memoryUsage      prometheus.Gauge
	tagCounters      *prometheus.GaugeVec
)

// InitMetrics initializes all Prometheus metrics
func InitMetrics() {
	// Create metrics
	totalMessages = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mailpit_messages",
		Help: "Total number of messages in the database",
	})

	unreadMessages = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mailpit_messages_unread",
		Help: "Number of unread messages in the database",
	})

	databaseSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mailpit_database_size_bytes",
		Help: "Size of the database in bytes",
	})

	messagesDeleted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailpit_messages_deleted_total",
		Help: "Total number of messages deleted",
	})

	smtpAccepted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailpit_smtp_accepted_total",
		Help: "Total number of SMTP messages accepted",
	})

	smtpRejected = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailpit_smtp_rejected_total",
		Help: "Total number of SMTP messages rejected",
	})

	smtpIgnored = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailpit_smtp_ignored_total",
		Help: "Total number of SMTP messages ignored (duplicates)",
	})

	smtpAcceptedSize = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailpit_smtp_accepted_size_bytes_total",
		Help: "Total size of accepted SMTP messages in bytes",
	})

	uptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mailpit_uptime_seconds",
		Help: "Uptime of Mailpit in seconds",
	})

	memoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mailpit_memory_usage_bytes",
		Help: "Memory usage in bytes",
	})

	tagCounters = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mailpit_tag_messages",
			Help: "Number of messages per tag",
		},
		[]string{"tag"},
	)

	// Register metrics
	Registry.MustRegister(totalMessages)
	Registry.MustRegister(unreadMessages)
	Registry.MustRegister(databaseSize)
	Registry.MustRegister(messagesDeleted)
	Registry.MustRegister(smtpAccepted)
	Registry.MustRegister(smtpRejected)
	Registry.MustRegister(smtpIgnored)
	Registry.MustRegister(smtpAcceptedSize)
	Registry.MustRegister(uptime)
	Registry.MustRegister(memoryUsage)
	Registry.MustRegister(tagCounters)
}

// UpdateMetrics updates all metrics with current values
func UpdateMetrics() {
	info := stats.Load()

	totalMessages.Set(float64(info.Messages))
	unreadMessages.Set(float64(info.Unread))
	databaseSize.Set(float64(info.DatabaseSize))
	messagesDeleted.Add(float64(info.RuntimeStats.MessagesDeleted))
	smtpAccepted.Add(float64(info.RuntimeStats.SMTPAccepted))
	smtpRejected.Add(float64(info.RuntimeStats.SMTPRejected))
	smtpIgnored.Add(float64(info.RuntimeStats.SMTPIgnored))
	smtpAcceptedSize.Add(float64(info.RuntimeStats.SMTPAcceptedSize))
	uptime.Set(float64(info.RuntimeStats.Uptime))
	memoryUsage.Set(float64(info.RuntimeStats.Memory))

	// Reset tag counters
	tagCounters.Reset()

	// Update tag counters
	for tag, count := range info.Tags {
		tagCounters.WithLabelValues(tag).Set(float64(count))
	}
}

// Returns the Prometheus handler & disables double compression in middleware
func GetHandler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{
		DisableCompression: true,
	})
}

// StartUpdater starts the periodic metrics update routine
func StartUpdater() {
	InitMetrics()
	UpdateMetrics()

	// Start periodic updates
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			UpdateMetrics()
		}
	}()
}

// StartSeparateServer starts a separate HTTP server for Prometheus metrics
func StartSeparateServer() {
	StartUpdater()

	logger.Log().Infof("[prometheus] metrics server listening on %s", config.PrometheusListen)

	// Create a dedicated mux for the metrics server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(Registry, promhttp.HandlerOpts{}))

	// Create a dedicated server instance
	server := &http.Server{
		Addr:    config.PrometheusListen,
		Handler: mux,
	}

	// Start HTTP server
	if err := server.ListenAndServe(); err != nil {
		logger.Log().Errorf("[prometheus] metrics server error: %s", err.Error())
	}
}

// GetMode returns the Prometheus run mode
func GetMode() string {
	mode := strings.ToLower(strings.TrimSpace(config.PrometheusListen))
	if mode == "false" {
		return "disabled"
	}
	if mode == "true" {
		return "integrated"
	}
	return "separate"
}
