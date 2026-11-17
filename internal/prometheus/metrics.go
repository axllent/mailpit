// Package prometheus provides Prometheus metrics for Mailpit
package prometheus

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/connlimit"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/stats"
)

// maxTagSeries is the most tags that are exported, each one as a separate series. Tags come from
// the content of messages, so without a limit whoever can send a message controls how many series
// (and so how much storage) a Prometheus server scraping this needs. The most used tags are
// exported, and mailpit_tags always reports how many there really are.
const maxTagSeries = 500

// Timeouts of the separate metrics server. They are variables so that the tests can shorten them.
var (
	serverReadTimeout  = 10 * time.Second
	serverWriteTimeout = 30 * time.Second
	serverIdleTimeout  = 60 * time.Second
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

// Replace sets all the values at once, so that a reader sees either the old values or the new
// ones, and never a mixture or none. The map must not be changed after this call.
func (v *gaugeVec) Replace(vals map[string]float64) {
	v.mu.Lock()
	v.vals = vals
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
	totalTags        = &gauge{}
	tagCounters      = newGaugeVec("tag")

	initOnce sync.Once
)

func register(name, help, typ string, g *gauge, vec *gaugeVec) {
	regMu.Lock()
	registry = append(registry, entry{name: name, help: help, typ: typ, g: g, vec: vec})
	regMu.Unlock()
}

// initMetrics registers the metrics. It only does so once, as registering them twice would
// output every metric twice, which is not valid.
func initMetrics() {
	initOnce.Do(registerMetrics)
}

func registerMetrics() {
	register("mailpit_database_size_bytes", "Size of the database in bytes", "gauge", databaseSize, nil)
	register("mailpit_memory_usage_bytes", "Memory usage in bytes", "gauge", memoryUsage, nil)
	register("mailpit_messages", "Total number of messages in the database", "gauge", totalMessages, nil)
	register("mailpit_messages_deleted_total", "Total number of messages deleted", "counter", messagesDeleted, nil)
	register("mailpit_messages_unread", "Number of unread messages in the database", "gauge", unreadMessages, nil)
	register("mailpit_smtp_accepted_size_bytes_total", "Total size of accepted SMTP messages in bytes", "counter", smtpAcceptedSize, nil)
	register("mailpit_smtp_accepted_total", "Total number of SMTP messages accepted", "counter", smtpAccepted, nil)
	register("mailpit_smtp_ignored_total", "Total number of SMTP messages ignored (duplicates)", "counter", smtpIgnored, nil)
	register("mailpit_smtp_rejected_total", "Total number of SMTP messages rejected", "counter", smtpRejected, nil)
	register("mailpit_tag_messages", fmt.Sprintf("Number of messages per tag, for the %d most used tags", maxTagSeries), "gauge", nil, tagCounters)
	register("mailpit_tags", "Total number of different tags", "gauge", totalTags, nil)
	register("mailpit_uptime_seconds", "Uptime of Mailpit in seconds", "gauge", uptime, nil)
}

func updateMetrics() {
	applyMetrics(stats.Load(false))
}

// applyMetrics sets the metrics from the application information.
func applyMetrics(info stats.AppInformation) {
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

	totalTags.Set(float64(len(info.Tags)))
	tagCounters.Replace(topTags(info.Tags, maxTagSeries))
}

// topTags returns the (up to) limit tags with the most messages. Tags with the same number of
// messages are ordered by name, so the same tags are returned each time.
func topTags(tags map[string]int64, limit int) map[string]float64 {
	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		if tags[names[i]] != tags[names[j]] {
			return tags[names[i]] > tags[names[j]]
		}

		return names[i] < names[j]
	})

	if len(names) > limit {
		names = names[:limit]
	}

	top := make(map[string]float64, len(names))
	for _, name := range names {
		top[name] = float64(tags[name])
	}

	return top
}

// safeUpdateMetrics updates the metrics, recovering from a panic so that a failure while doing
// so cannot stop Mailpit.
func safeUpdateMetrics() {
	defer func() {
		if r := recover(); r != nil {
			logger.Log().Errorf("[prometheus] recovered from panic updating metrics: %v", r)
		}
	}()

	updateMetrics()
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
	safeUpdateMetrics()

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			safeUpdateMetrics()
		}
	}()
}

// StartSeparateServer starts a separate HTTP server for Prometheus metrics
func StartSeparateServer() {
	StartUpdater()

	ln, err := net.Listen("tcp", config.PrometheusListen)
	if err != nil {
		logger.Log().Errorf("[prometheus] metrics server error: %s", err.Error())
		return
	}

	logger.Log().Infof("[prometheus] metrics server listening on %s", config.PrometheusListen)

	if err := serve(ln, connlimit.Max); err != nil {
		logger.Log().Errorf("[prometheus] metrics server error: %s", err.Error())
	}
}

// serve serves the metrics on the listener, allowing at most maxConns concurrent connections
// like Mailpit's other network listeners.
func serve(ln net.Listener, maxConns int) error {
	return newServer().Serve(connlimit.Listener("prometheus", ln, maxConns))
}

// newServer returns the HTTP server for the separate metrics listener. Every phase has a
// timeout, so a client cannot hold a connection (and a file descriptor) open indefinitely.
func newServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", GetHandler())

	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
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
