package metrics

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/yourusername/platypus/internal/models"
)

type CollectorConfig struct {
    RetentionPeriod   time.Duration
    CollectionInterval time.Duration
    BatchSize         int
    BufferSize        int
}

type Collector struct {
    config  CollectorConfig
    metrics map[string]*ServerMetrics
    buffer  chan MetricBatch
    mu      sync.RWMutex

    // Prometheus метрики
    powerUsageGauge    *prometheus.GaugeVec
    carbonFootprintGauge *prometheus.GaugeVec
    cpuUsageGauge      *prometheus.GaugeVec
    memoryUsageGauge   *prometheus.GaugeVec
}

type ServerMetrics struct {
    Data      []models.MetricData
    LastUpdate time.Time
}

type MetricBatch struct {
    ServerID string
    Metrics  []models.MetricData
    Timestamp time.Time
}

func NewCollector(config CollectorConfig) *Collector {
    c := &Collector{
        config:  config,
        metrics: make(map[string]*ServerMetrics),
        buffer:  make(chan MetricBatch, config.BufferSize),
    }

    // Инициализация Prometheus метрик
    c.initPrometheusMetrics()

    return c
}

func (c *Collector) initPrometheusMetrics() {
    c.powerUsageGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "server_power_usage_watts",
            Help: "Current power usage in watts",
        },
        []string{"server_id", "region"},
    )

    c.carbonFootprintGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "server_carbon_footprint_kg",
            Help: "Current carbon footprint in kg CO2",
        },
        []string{"server_id", "region"},
    )

    c.cpuUsageGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "server_cpu_usage_percent",
            Help: "Current CPU usage percentage",
        },
        []string{"server_id", "region"},
    )

    c.memoryUsageGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "server_memory_usage_percent",
            Help: "Current memory usage percentage",
        },
        []string{"server_id", "region"},
    )

    // Регистрация метрик в Prometheus
    prometheus.MustRegister(
        c.powerUsageGauge,
        c.carbonFootprintGauge,
        c.cpuUsageGauge,
        c.memoryUsageGauge,
    )
}

func (c *Collector) Start(ctx context.Context) error {
    // Запускаем обработчик буфера метрик
    go c.processBuffer(ctx)
    
    // Запускаем очистку старых метрик
    go c.cleanupOldMetrics(ctx)

    return nil
}

func (c *Collector) processBuffer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case batch := <-c.buffer:
            c.processBatch(batch)
        }
    }
}

func (c *Collector) processBatch(batch MetricBatch) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if _, exists := c.metrics[batch.ServerID]; !exists {
        c.metrics[batch.ServerID] = &ServerMetrics{
            Data: make([]models.MetricData, 0),
        }
    }

    // Добавляем новые метрики
    c.metrics[batch.ServerID].Data = append(c.metrics[batch.ServerID].Data, batch.Metrics...)
    c.metrics[batch.ServerID].LastUpdate = batch.Timestamp

    // Обновляем Prometheus метрики
    for _, metric := range batch.Metrics {
        labels := prometheus.Labels{"server_id": batch.ServerID, "region": "default"}
        
        c.powerUsageGauge.With(labels).Set(metric.PowerUsage)
        c.carbonFootprintGauge.With(labels).Set(metric.CarbonFootprint)
        c.cpuUsageGauge.With(labels).Set(metric.CPUUsage)
        c.memoryUsageGauge.With(labels).Set(metric.MemoryUsage)
    }
}

func (c *Collector) CollectMetrics(serverID string, data models.MetricData) error {
    batch := MetricBatch{
        ServerID:  serverID,
        Metrics:   []models.MetricData{data},
        Timestamp: time.Now(),
    }

    select {
    case c.buffer <- batch:
        return nil
    default:
        return fmt.Errorf("metric buffer is full")
    }
}

func (c *Collector) GetMetrics(serverID string) ([]models.MetricData, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if metrics, exists := c.metrics[serverID]; exists {
        return metrics.Data, nil
    }
    return nil, fmt.Errorf("no metrics found for server: %s", serverID)
}

func (c *Collector) cleanupOldMetrics(ctx context.Context) {
    ticker := time.NewTicker(c.config.CollectionInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.mu.Lock()
            cutoff := time.Now().Add(-c.config.RetentionPeriod)
            
            for serverID, serverMetrics := range c.metrics {
                filtered := make([]models.MetricData, 0)
                for _, metric := range serverMetrics.Data {
                    if time.Unix(metric.Timestamp, 0).After(cutoff) {
                        filtered = append(filtered, metric)
                    }
                }
                c.metrics[serverID].Data = filtered
            }
            c.mu.Unlock()
        }
    }
} 