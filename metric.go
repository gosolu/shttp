package shttp

import "github.com/prometheus/client_golang/prometheus"

var requestLabels = []string{
	"method",
	"host",
	"path",
	"code",
	"spend",
}

var (
	httpRequestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "solu",
		Subsystem: "http",
		Name:      "request_counter",
		Help:      "http request counter",
	}, requestLabels)

	httpErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "solu",
		Subsystem: "http",
		Name:      "error_counter",
		Help:      "http request error counter",
	}, requestLabels)
)

func init() {
	prometheus.MustRegister(
		httpRequestCounter,
		httpErrorCounter,
	)
}
