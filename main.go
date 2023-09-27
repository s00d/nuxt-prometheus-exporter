package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type RequestBody struct {
	Route    string `json:"route"`
	Code     string `json:"code"`
	Method   string `json:"method"`
	Date     string `json:"date"`
	Duration string `json:"duration"`
}

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nuxt_requests_total",
			Help: "Total number of requests processed by Node.js",
		},
		[]string{"route", "code", "method", "date"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "requests_duration_seconds",
			Help:    "Time taken to process request",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method"},
	)
)

var customRegistry = prometheus.NewRegistry()

func main() {
	// Получаем порты из аргументов командной строки
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)

	metricsPort := "45555"
	clean := false
	if len(os.Args) >= 2 {
		metricsPort = os.Args[1]
	}
	if len(os.Args) >= 3 && os.Args[2] == "clean" {
		clean = true
	}

	// Регистрируем метрику в Prometheus
	customRegistry.MustRegister(requestsTotal, requestDuration)

	// Запускаем HTTP-сервер для сбора метрик Prometheus
	//http.Handle("/metrics", customMetricsHandler())
	//http.Handle("/metrics", promhttp.Handler())
	//http.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		// Очистка после обработки запроса
		if clean {
			requestsTotal.Reset()
			requestDuration.Reset()
		}
	})

	landingConfig := web.LandingConfig{
		Name:        "Noxt Prometheus Exporter",
		Description: "Prometheus Exporter for Noxt",
		HeaderColor: "#039900",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: "/metrics",
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	http.Handle("/", landingPage)

	http.HandleFunc("/nodejs-requests", func(w http.ResponseWriter, r *http.Request) {
		var p RequestBody
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		duration, err := strconv.ParseFloat(p.Duration, 64)
		if err != nil {
			http.Error(w, "Invalid duration value", http.StatusBadRequest)
			return
		}

		timestamp, err := strconv.ParseInt(p.Date, 10, 64)
		if err != nil {
			http.Error(w, "Invalid parse date", http.StatusBadRequest)
			return
		}

		date := time.Unix(timestamp, 0).Format(time.RFC3339)

		requestDuration.With(prometheus.Labels{"route": p.Route, "method": p.Method}).Observe(duration)
		requestsTotal.With(prometheus.Labels{"route": p.Route, "code": fmt.Sprint(p.Code), "method": p.Method, "date": date}).Inc()
		w.WriteHeader(http.StatusOK)
	})

	http.ListenAndServe(":"+metricsPort, nil)

	go func() {
		if err := http.ListenAndServe(":"+metricsPort, nil); err != nil {
			fmt.Println("Failed to setup listener: ", err.Error())
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
