package instrumentation

import (
	"github.com/prometheus/client_golang/prometheus"
)

var Stats = new(Collectors)

type Collectors struct {
	GatewaySessionsGauge     *prometheus.GaugeVec
	RoomParticipantsGauge    *prometheus.GaugeVec
	RequestDurationHistogram *prometheus.HistogramVec
}

func (c *Collectors) Init() {
	c.GatewaySessionsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "galaxy",
		Subsystem: "gateways",
		Name:      "sessions",
		Help:      "WebRTC Gateways active sessions",
	}, []string{
		// gateway name
		"name",
		// gateway type (rooms, streaming)
		"type"})

	c.RoomParticipantsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "galaxy",
		Subsystem: "api",
		Name:      "participants",
		Help:      "Active room participants",
	}, []string{
		// room name
		"name",
	})

	c.RequestDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "galaxy",
		Subsystem: "api",
		Name:      "request_duration",
		Help:      "Time (in milliseconds) spent serving HTTP requests.",
	}, []string{"method", "route", "status_code"})

	prometheus.MustRegister(c.GatewaySessionsGauge)
	prometheus.MustRegister(c.RoomParticipantsGauge)
	prometheus.MustRegister(c.RequestDurationHistogram)
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())
}

func (c *Collectors) Reset() {
	c.GatewaySessionsGauge.Reset()
	c.RoomParticipantsGauge.Reset()
	c.RequestDurationHistogram.Reset()
}
