package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "okteto_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"status", "route"})
)

func buildMetricsHandler(handler http.Handler) http.Handler {
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "okteto_in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "okteto_api_requests_total",
			Help: "A counter for requests to the wrapped handler.",
		},
		[]string{"code", "method"},
	)

	// responseSize has no labels, making it a zero-dimensional
	// ObserverVec.
	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "okteto_response_size_bytes",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{},
	)

	prometheus.MustRegister(inFlightGauge, counter, duration, responseSize)

	metricsChain := promhttp.InstrumentHandlerInFlight(inFlightGauge,
		promhttp.InstrumentHandlerCounter(counter,
			promhttp.InstrumentHandlerResponseSize(responseSize, handler),
		),
	)

	return metricsChain
}
