package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"io/ioutil"
	"net/http"
	"os"
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
		[]string{"route", "code", "method", "date", "duration"},
	)
)

func main() {
	// Получаем порты из аргументов командной строки
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)

	metricsPort := "45555"
	if len(os.Args) >= 2 {
		metricsPort = os.Args[1]
	}

	// Регистрируем метрику в Prometheus
	prometheus.MustRegister(requestsTotal)

	// Запускаем HTTP-сервер для сбора метрик Prometheus
	http.Handle("/metrics", customMetricsHandler())
	http.HandleFunc("/nodejs-requests", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Обработка и сохранение информации о запросе
		processRequest(body)

		// Возвращаем успешный ответ
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		err := http.ListenAndServe(":"+metricsPort, nil)
		if err != nil {
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
	}()

	// Ожидаем завершения работы
	select {}
}

func processRequest(requestBody []byte) {
	var requestData RequestBody
	err := json.Unmarshal(requestBody, &requestData)
	if err != nil {
		fmt.Println("Failed to unmarshal request body:", err)
		return
	}

	// Увеличиваем счетчик с метками
	requestsTotal.WithLabelValues(requestData.Route, requestData.Code, requestData.Method, requestData.Date, requestData.Duration).Inc()
}

func customMetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		registry.MustRegister(requestsTotal)
		gatherers := prometheus.Gatherers{
			registry,
		}
		handler := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	})
}
