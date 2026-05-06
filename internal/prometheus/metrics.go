// Package prometheus provides Prometheus metrics for Mailpit
package prometheus

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/stats"
)

type gauge struct {
	mu  sync.RWMutex
	val float64
}

func (g *gauge) Set(v float64) {
	g.mu.Lock()
	g.val = v
	g.mu.Unlock()
}

func (g *gauge) get() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.val
}

type gaugeVec struct {
	mu    sync.RWMutex
	label string
	vals  map[string]float64
}

func newGaugeVec(label string) *gaugeVec {
	return &gaugeVec{label: label, vals: make(map[string]float64)}
}

func (v *gaugeVec) Set(labelVal string, val float64) {
	v.mu.Lock()
	v.vals[labelVal] = val
	v.mu.Unlock()
}

func (v *gaugeVec) Reset() {
	v.mu.Lock()
	v.vals = make(map[string]float64)
	v.mu.Unlock()
}

type entry struct {
	name string
	help string
	typ  string
	g    *gauge
	vec  *gaugeVec
}

var (
	regMu    sync.RWMutex
	registry []entry

	totalMessages    = &gauge{}
	unreadMessages   = &gauge{}
	databaseSize     = &gauge{}
	messagesDeleted  = &gauge{}
	smtpAccepted     = &gauge{}
	smtpRejected     = &gauge{}
	smtpIgnored      = &gauge{}
	smtpAcceptedSize = &gauge{}
	uptime           = &gauge{}
	memoryUsage      = &gauge{}
	tagCounters      = newGaugeVec("tag")
)

func register(name, help, typ string, g *gauge, vec *gaugeVec) {
	regMu.Lock()
	registry = append(registry, entry{name: name, help: help, typ: typ, g: g, vec: vec})
	regMu.Unlock()
}

func initMetrics() {
	register("mailpit_database_size_bytes", "Size of the database in bytes", "gauge", databaseSize, nil)
	register("mailpit_memory_usage_bytes", "Memory usage in bytes", "gauge", memoryUsage, nil)
	register("mailpit_messages", "Total number of messages in the database", "gauge", totalMessages, nil)
	register("mailpit_messages_deleted_total", "Total number of messages deleted", "counter", messagesDeleted, nil)
	register("mailpit_messages_unread", "Number of unread messages in the database", "gauge", unreadMessages, nil)
	register("mailpit_smtp_accepted_size_bytes_total", "Total size of accepted SMTP messages in bytes", "counter", smtpAcceptedSize, nil)
	register("mailpit_smtp_accepted_total", "Total number of SMTP messages accepted", "counter", smtpAccepted, nil)
	register("mailpit_smtp_ignored_total", "Total number of SMTP messages ignored (duplicates)", "counter", smtpIgnored, nil)
	register("mailpit_smtp_rejected_total", "Total number of SMTP messages rejected", "counter", smtpRejected, nil)
	register("mailpit_tag_messages", "Number of messages per tag", "gauge", nil, tagCounters)
	register("mailpit_uptime_seconds", "Uptime of Mailpit in seconds", "gauge", uptime, nil)
}

func updateMetrics() {
	info := stats.Load(false)

	totalMessages.Set(float64(info.Messages))
	unreadMessages.Set(float64(info.Unread))
	databaseSize.Set(float64(info.DatabaseSize))
	messagesDeleted.Set(float64(info.RuntimeStats.MessagesDeleted))
	smtpAccepted.Set(float64(info.RuntimeStats.SMTPAccepted))
	smtpRejected.Set(float64(info.RuntimeStats.SMTPRejected))
	smtpIgnored.Set(float64(info.RuntimeStats.SMTPIgnored))
	smtpAcceptedSize.Set(float64(info.RuntimeStats.SMTPAcceptedSize))
	uptime.Set(float64(info.RuntimeStats.Uptime))
	memoryUsage.Set(float64(info.RuntimeStats.Memory))

	tagCounters.Reset()
	for tag, count := range info.Tags {
		tagCounters.Set(tag, float64(count))
	}
}

func writeMetrics(w io.Writer) {
	regMu.RLock()
	entries := make([]entry, len(registry))
	copy(entries, registry)
	regMu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	for _, e := range entries {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", e.name, e.help, e.name, e.typ)
		if e.g != nil {
			fmt.Fprintf(w, "%s %s\n", e.name, formatFloat(e.g.get()))
		} else {
			e.vec.mu.RLock()
			keys := make([]string, 0, len(e.vec.vals))
			snapshot := make(map[string]float64, len(e.vec.vals))
			for k, v := range e.vec.vals {
				keys = append(keys, k)
				snapshot[k] = v
			}
			e.vec.mu.RUnlock()
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(w, "%s{%s=\"%s\"} %s\n", e.name, e.vec.label, escapeLabelValue(k), formatFloat(snapshot[k]))
			}
		}
	}
}

func escapeLabelValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// GetHandler returns the Prometheus metrics HTTP handler
func GetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		writeMetrics(w)
	})
}

// StartUpdater starts the periodic metrics update routine
func StartUpdater() {
	initMetrics()
	updateMetrics()

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateMetrics()
		}
	}()
}

// StartSeparateServer starts a separate HTTP server for Prometheus metrics
func StartSeparateServer() {
	StartUpdater()

	logger.Log().Infof("[prometheus] metrics server listening on %s", config.PrometheusListen)

	mux := http.NewServeMux()
	mux.Handle("/metrics", GetHandler())

	server := &http.Server{
		Addr:              config.PrometheusListen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Log().Errorf("[prometheus] metrics server error: %s", err.Error())
	}
}

// GetMode returns the Prometheus run mode
func GetMode() string {
	mode := strings.ToLower(strings.TrimSpace(config.PrometheusListen))
	switch mode {
	case "false", "":
		return "disabled"
	case "true":
		return "integrated"
	default:
		return "separate"
	}
}
