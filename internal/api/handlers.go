package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/YumeNoTenshi/platypus/internal/metrics"
	"github.com/YumeNoTenshi/platypus/internal/models"
)

type Server struct {
	collector *metrics.Collector
	analyzer  *metrics.Analyzer
}

func (s *Server) Router() *mux.Router {
	r := mux.NewRouter()
	
	// Добавляем middleware для всех маршрутов
	r.Use(LoggingMiddleware)
	
	// API версия v1
	v1 := r.PathPrefix("/api/v1").Subrouter()
	
	// Применяем аутентификацию ко всем маршрутам, кроме /health
	protected := v1.NewRoute().Subrouter()
	protected.Use(AuthMiddleware)
	
	// Открытые маршруты
	v1.HandleFunc("/health", s.handleHealth).Methods("GET")
 
	// Защищенные маршруты
	protected.HandleFunc("/metrics", s.handleGetMetrics).Methods("GET")
	protected.HandleFunc("/metrics", s.handlePostMetrics).Methods("POST")
	protected.HandleFunc("/servers", s.handleGetServers).Methods("GET")
	protected.HandleFunc("/servers/{id}", s.handleGetServer).Methods("GET")
	protected.HandleFunc("/eco-score", s.handleGetEcoScore).Methods("POST")
	protected.HandleFunc("/eco-tags", s.handleGetEcoTags).Methods("GET")
	protected.HandleFunc("/status", s.handleStatus).Methods("GET")
	
	return r
}

func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		respondWithError(w, http.StatusBadRequest, "server_id is required")
		return
	}

	metrics, err := s.collector.GetMetrics(serverID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, MetricResponse{
		Status: "success",
		Data:   metrics,
	})
}

func (s *Server) handlePostMetrics(w http.ResponseWriter, r *http.Request) {
	var metricData models.MetricData
	if err := json.NewDecoder(r.Body).Decode(&metricData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	metricData.Timestamp = time.Now().Unix()
	
	if err := s.collector.CollectMetrics(metricData.ServerID, metricData); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"status": "success",
		"message": "Metrics collected successfully",
	})
}

func (s *Server) handleGetEcoScore(w http.ResponseWriter, r *http.Request) {
	var req EcoScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	metrics, err := s.collector.GetMetrics(req.ServerID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	score := s.analyzer.CalculateEcoScore(metrics)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"server_id": req.ServerID,
			"eco_score": score,
			"period": req.Period,
		},
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"error","message":"Error marshaling JSON"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
} 