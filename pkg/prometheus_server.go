package pkg

import (
	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	activeSubscribes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "Broker",
			Subsystem: "broker-master",
			Name:      "active_subscribes",
			Help:      "active subscribes on master broker",
		}, []string{"subject"})

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "Broker",
			Subsystem: "broker-master",
			Name:      "requests_duration",
			Help:      "requestsDuration of master broker",
			Buckets:   []float64{25, 50, 95, 99},
		}, []string{"method_name", "subject", "successful"})

	methodCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "Broker",
			Subsystem: "master-broker",
			Name:      "grpc_method_count",
			Help:      "grpc method count",
		}, []string{"method_name", "subject"})

	errorRate = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "Broker",
		Subsystem: "master-broker",
		Name:      "error_rate",
		Help:      "error rate of master broker",
		Buckets:   []float64{25, 50, 70, 80, 90, 95, 99},
	}, []string{"method_name", "subject", "error"})
)

func GetErrorRateMetric() *prometheus.HistogramVec {
	return errorRate
}
func GetMethodCountMetric() *prometheus.GaugeVec {
	return methodCount
}
func GetMethodDurationMetric() *prometheus.HistogramVec {
	return requestDuration
}
func GetActiveSubscribesMetric() *prometheus.GaugeVec {
	return activeSubscribes
}

func init() {
	prometheus.MustRegister(methodCount)
	prometheus.MustRegister(errorRate)
	prometheus.MustRegister(activeSubscribes)
	prometheus.MustRegister(requestDuration)
	log.Info("prometheus registered successfully!")
}

func StartPrometheusServer() {
	go func() {
		port := ":5000"
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(port, nil)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("can't start prometheus!")
			panic("can't start prometheus!")
		}
		log.Info("prometheus start on port: ", port)
	}()

}
