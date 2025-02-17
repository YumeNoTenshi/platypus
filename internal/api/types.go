package api

import (
	"github.com/yourusername/platypus/internal/metrics"
	"github.com/yourusername/platypus/internal/models"
)

type Server struct {
	collector *metrics.Collector
	analyzer  *metrics.Analyzer
}

type MetricResponse struct {
	Status string         `json:"status"`
	Data   []models.MetricData `json:"data"`
}

type ServerResponse struct {
	Status string         `json:"status"`
	Data   []models.Server `json:"data"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type EcoScoreRequest struct {
	ServerID string `json:"server_id"`
	Period   string `json:"period"` // "1h", "24h", "7d"
} 