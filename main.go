// http-server-projeto-korp
// Servidor HTTP do Projeto Korp.
// Expõe /projeto-korp, que retorna o horário UTC, /health, que verifica a
// disponibilidade, e /metrics, que publica métricas para o Prometheus.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// requestMetrics guarda os dados agregados de cada requisicao HTTP.
// O mutex protege os mapas e o histograma contra acessos concorrentes.
var requestMetrics = struct {
	sync.Mutex
	total          uint64
	byStatus       map[int]uint64
	durationCount  uint64
	durationSum    float64
	durationBucket map[float64]uint64
}{
	byStatus:       make(map[int]uint64),
	durationBucket: make(map[float64]uint64),
}

// metricsHandler publica as métricas de monitoramento no formato Prometheus.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Define o tipo de conteúdo esperado pelo Prometheus.
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Copia os valores protegidos para evitar manter o mutex durante a resposta.
	requestMetrics.Lock()
	total := requestMetrics.total
	byStatus := make(map[int]uint64, len(requestMetrics.byStatus))
	for status, count := range requestMetrics.byStatus {
		byStatus[status] = count
	}
	durationCount := requestMetrics.durationCount
	durationSum := requestMetrics.durationSum
	durationBucket := make(map[float64]uint64, len(requestMetrics.durationBucket))
	for bucket, count := range requestMetrics.durationBucket {
		durationBucket[bucket] = count
	}
	requestMetrics.Unlock()

	// Publica o contador acumulado de requisições.
	fmt.Fprintln(w, "# HELP http_requests_total Total de requisicoes recebidas pelo servico.")
	fmt.Fprintln(w, "# TYPE http_requests_total counter")
	fmt.Fprintf(w, "http_requests_total %d\n", total)

	// Publica a quantidade de respostas agrupada pelo status HTTP.
	fmt.Fprintln(w, "# HELP http_responses_total Total de respostas por status HTTP.")
	fmt.Fprintln(w, "# TYPE http_responses_total counter")
	for status, count := range byStatus {
		fmt.Fprintf(w, "http_responses_total{status=\"%d\"} %d\n", status, count)
	}

	// Publica o histograma de duração em segundos, incluindo soma e quantidade.
	fmt.Fprintln(w, "# HELP http_request_duration_seconds Duracao das requisicoes HTTP em segundos.")
	fmt.Fprintln(w, "# TYPE http_request_duration_seconds histogram")
	for _, bucket := range []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10} {
		fmt.Fprintf(w, "http_request_duration_seconds_bucket{le=\"%g\"} %d\n", bucket, durationBucket[bucket])
	}
	fmt.Fprintf(w, "http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", durationCount)
	fmt.Fprintf(w, "http_request_duration_seconds_sum %f\n", durationSum)
	fmt.Fprintf(w, "http_request_duration_seconds_count %d\n", durationCount)

	// Publica o indicador de disponibilidade do processo HTTP.
}

// healthHandler responde às verificações de saúde do serviço.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Informa que a resposta será um objeto JSON.
	w.Header().Set("Content-Type", "application/json")

	// O status HTTP 200 confirma que o processo está respondendo.
	fmt.Fprint(w, `{"status":"ok"}`)
}

// countRequests envolve um handler e contabiliza cada chamada antes de
// delegar o processamento para o handler original.
func countRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// O Prometheus consulta /metrics repetidamente; essa consulta técnica
		// não representa tráfego da aplicação e não entra no contador.
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Mede o tempo total incluindo a execução do handler de negócio.
		startedAt := time.Now()
		responseWriter := &statusRecorder{ResponseWriter: w}

		// Preserva o comportamento do handler recebido como parâmetro.
		next.ServeHTTP(responseWriter, r)

		// Registra contagem, status e duração depois que a resposta terminou.
		duration := time.Since(startedAt).Seconds()
		status := responseWriter.status
		if status == 0 {
			status = http.StatusOK
		}
		requestMetrics.Lock()
		requestMetrics.total++
		requestMetrics.byStatus[status]++
		requestMetrics.durationCount++
		requestMetrics.durationSum += duration
		for _, bucket := range []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10} {
			if duration <= bucket {
				requestMetrics.durationBucket[bucket]++
			}
		}
		requestMetrics.Unlock()
	})
}

// statusRecorder captura o status HTTP escrito pelo handler encapsulado.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader guarda o status antes de enviá-lo ao cliente.
func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

// Write garante status 200 quando o handler escreve sem chamar WriteHeader.
func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder.ResponseWriter.Write(body)
}

// projetoKorpHandler é a função que trata as requisições feitas para
// o endpoint "/projeto-korp".
// Toda vez que alguém acessa esse endereço, essa função é chamada.
func projetoKorpHandler(w http.ResponseWriter, r *http.Request) {

	// time.Now() pega o horário atual da máquina.
	// .UTC() converte esse horário para o padrão UTC.
	// Isso é feito aqui dentro da função para que o horário seja
	// sempre calculado no momento exato da requisição.
	horarioAtual := time.Now().UTC().Format("15:04:05")

	// Cria um map que será convertido em um objeto JSON na resposta.
	resposta := map[string]string{
		"nome":    "Projeto Korp",
		"horario": horarioAtual,
	}

	// Define no cabeçalho da resposta que o conteúdo enviado é um JSON.
	w.Header().Set("Content-Type", "application/json")

	// Codifica o map como JSON e escreve no corpo da resposta HTTP.
	json.NewEncoder(w).Encode(resposta)
}

// main é a função principal, o ponto de partida do programa.
func main() {

	// Cria um roteador local e registra os endpoints da aplicação.
	mux := http.NewServeMux()
	mux.HandleFunc("/projeto-korp", projetoKorpHandler)
	mux.HandleFunc("/metrics", metricsHandler)
	mux.HandleFunc("/health", healthHandler)

	fmt.Println("Servidor http-server-projeto-korp rodando em http://localhost:8080/projeto-korp")

	// Inicia o servidor na porta 8080 usando o roteador configurado acima.
	http.ListenAndServe(":8080", countRequests(mux))
}
