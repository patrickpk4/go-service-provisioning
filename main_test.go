// Testes unitarios dos handlers HTTP e do middleware de metricas do Projeto Korp.
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resetRequestMetrics() {
	requestMetrics.Lock()
	requestMetrics.total = 0
	requestMetrics.byStatus = make(map[int]uint64)
	requestMetrics.durationCount = 0
	requestMetrics.durationSum = 0
	requestMetrics.durationBucket = make(map[float64]uint64)
	requestMetrics.Unlock()
}

func TestProjetoKorpHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/projeto-korp", nil)

	projetoKorpHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status esperado 200, recebido %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type inesperado: %q", recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Body.String(), `"nome":"Projeto Korp"`) {
		t.Fatalf("resposta inesperada: %s", recorder.Body.String())
	}
}

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	healthHandler(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"status":"ok"}` {
		t.Fatalf("resposta de health inesperada: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCountRequestsRecordsStatusAndDuration(t *testing.T) {
	resetRequestMetrics()
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {})

	recorder := httptest.NewRecorder()
	countRequests(mux).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok", nil))

	requestMetrics.Lock()
	defer requestMetrics.Unlock()
	if requestMetrics.total != 1 || requestMetrics.byStatus[200] != 1 {
		t.Fatalf("metricas de requisicao inesperadas: total=%d status200=%d", requestMetrics.total, requestMetrics.byStatus[200])
	}
	if requestMetrics.durationCount != 1 || requestMetrics.durationSum <= 0 {
		t.Fatalf("duracao nao registrada: count=%d sum=%f", requestMetrics.durationCount, requestMetrics.durationSum)
	}
}
