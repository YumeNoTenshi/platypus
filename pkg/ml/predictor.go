package ml

import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "sync"
    "time"
    
    "github.com/yourusername/platypus/internal/metrics"
    "github.com/yourusername/platypus/internal/models"
    "gonum.org/v1/gonum/stat"
)

// Prediction представляет прогноз для сервера
type Prediction struct {
    ServerID        string    `json:"server_id"`
    Timestamp       time.Time `json:"timestamp"`
    CPUUsage        float64   `json:"cpu_usage"`
    MemoryUsage     float64   `json:"memory_usage"`
    PowerUsage      float64   `json:"power_usage"`
    CarbonFootprint float64   `json:"carbon_footprint"`
    Confidence      float64   `json:"confidence"` // 0-1, где 1 - максимальная уверенность
}

type PredictorConfig struct {
    HistoryWindow    time.Duration // Окно исторических данных для анализа
    PredictionWindow time.Duration // Окно прогнозирования
    UpdateInterval   time.Duration // Интервал обновления моделей
    MinDataPoints    int          // Минимальное количество точек для прогноза
    ModelPath        string       // Путь к сохраненным моделям
}

type Predictor struct {
    config    PredictorConfig
    collector *metrics.Collector
    models    map[string]*TimeSeriesModel // ServerID -> Model
    mu        sync.RWMutex
}

// TimeSeriesModel представляет модель временного ряда для одного сервера
type TimeSeriesModel struct {
    ServerID     string
    Coefficients []float64    // Коэффициенты модели
    LastUpdate   time.Time    // Время последнего обновления
    Seasonality  time.Duration // Период сезонности (например, 24 часа)
    Trends       []Trend      // Обнаруженные тренды
}

type Trend struct {
    StartTime time.Time
    EndTime   time.Time
    Slope     float64
    Type      TrendType
}

type TrendType string

const (
    TrendIncreasing TrendType = "increasing"
    TrendDecreasing TrendType = "decreasing"
    TrendStable     TrendType = "stable"
)

func NewPredictor(config PredictorConfig, collector *metrics.Collector) *Predictor {
    return &Predictor{
        config:    config,
        collector: collector,
        models:    make(map[string]*TimeSeriesModel),
    }
}

func (p *Predictor) Start(ctx context.Context) error {
    // Загружаем сохраненные модели
    if err := p.loadModels(); err != nil {
        return err
    }

    ticker := time.NewTicker(p.config.UpdateInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := p.updateModels(ctx); err != nil {
                continue
            }
            if err := p.saveModels(); err != nil {
                continue
            }
        }
    }
}

func (p *Predictor) PredictServerMetrics(ctx context.Context, serverID string, horizon time.Duration) ([]Prediction, error) {
    p.mu.RLock()
    model, exists := p.models[serverID]
    p.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("no model found for server %s", serverID)
    }

    // Получаем последние метрики для начальной точки прогноза
    metrics, err := p.collector.GetMetrics(serverID)
    if err != nil {
        return nil, err
    }

    if len(metrics) < p.config.MinDataPoints {
        return nil, fmt.Errorf("insufficient data points for prediction")
    }

    // Создаем прогнозы на заданный период
    predictions := make([]Prediction, 0)
    currentTime := time.Now()
    interval := time.Hour // Интервал между прогнозами

    for t := currentTime; t.Before(currentTime.Add(horizon)); t = t.Add(interval) {
        prediction := p.generatePrediction(model, metrics, t)
        predictions = append(predictions, prediction)
    }

    return predictions, nil
}

func (p *Predictor) generatePrediction(model *TimeSeriesModel, historicalData []models.MetricData, targetTime time.Time) Prediction {
    // Применяем сезонную декомпозицию
    seasonal := p.calculateSeasonalComponent(historicalData, targetTime)
    
    // Вычисляем тренд
    trend := p.calculateTrendComponent(model.Trends, targetTime)
    
    // Получаем последние актуальные данные
    latest := historicalData[len(historicalData)-1]
    
    // Комбинируем компоненты для прогноза
    prediction := Prediction{
        ServerID:        model.ServerID,
        Timestamp:       targetTime,
        CPUUsage:        math.Max(0, latest.CPUUsage * (1 + trend) + seasonal),
        PowerUsage:      math.Max(0, latest.PowerUsage * (1 + trend) + seasonal),
        MemoryUsage:     math.Max(0, latest.MemoryUsage * (1 + trend) + seasonal),
        CarbonFootprint: latest.CarbonFootprint * (1 + trend),
    }
    
    // Рассчитываем уверенность в прогнозе
    prediction.Confidence = p.calculateConfidence(historicalData, prediction)
    
    return prediction
}

func (p *Predictor) updateModels(ctx context.Context) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Получаем список всех серверов
    servers, err := p.getActiveServers(ctx)
    if err != nil {
        return err
    }

    for _, serverID := range servers {
        // Получаем исторические данные
        metrics, err := p.collector.GetMetrics(serverID)
        if err != nil {
            continue
        }

        if len(metrics) < p.config.MinDataPoints {
            continue
        }

        // Обновляем или создаем модель
        model := p.createTimeSeriesModel(serverID, metrics)
        p.models[serverID] = model
    }

    return nil
}

func (p *Predictor) createTimeSeriesModel(serverID string, data []models.MetricData) *TimeSeriesModel {
    // Извлекаем временные ряды
    times := make([]float64, len(data))
    cpuValues := make([]float64, len(data))
    powerValues := make([]float64, len(data))

    for i, d := range data {
        times[i] = float64(d.Timestamp)
        cpuValues[i] = d.CPUUsage
        powerValues[i] = d.PowerUsage
    }

    // Находим коэффициенты регрессии
    var coefficients []float64
    alpha, beta := stat.LinearRegression(times, cpuValues, nil, false)
    coefficients = append(coefficients, alpha, beta)

    // Определяем сезонность
    seasonality := p.detectSeasonality(data)

    // Определяем тренды
    trends := p.detectTrends(data)

    return &TimeSeriesModel{
        ServerID:     serverID,
        Coefficients: coefficients,
        LastUpdate:   time.Now(),
        Seasonality:  seasonality,
        Trends:       trends,
    }
}

func (p *Predictor) detectSeasonality(data []models.MetricData) time.Duration {
    // Анализируем автокорреляцию для определения сезонности
    // Упрощенная версия - проверяем суточную сезонность
    return 24 * time.Hour
}

func (p *Predictor) detectTrends(data []models.MetricData) []Trend {
    var trends []Trend
    windowSize := 12 // Размер окна для определения тренда

    for i := 0; i < len(data)-windowSize; i += windowSize {
        window := data[i:i+windowSize]
        slope := p.calculateSlope(window)
        
        trend := Trend{
            StartTime: time.Unix(window[0].Timestamp, 0),
            EndTime:   time.Unix(window[len(window)-1].Timestamp, 0),
            Slope:     slope,
            Type:      p.classifyTrend(slope),
        }
        
        trends = append(trends, trend)
    }

    return trends
}

func (p *Predictor) calculateSlope(data []models.MetricData) float64 {
    x := make([]float64, len(data))
    y := make([]float64, len(data))
    
    for i, d := range data {
        x[i] = float64(i)
        y[i] = d.PowerUsage
    }
    
    _, slope := stat.LinearRegression(x, y, nil, false)
    return slope
}

func (p *Predictor) classifyTrend(slope float64) TrendType {
    threshold := 0.1
    if slope > threshold {
        return TrendIncreasing
    } else if slope < -threshold {
        return TrendDecreasing
    }
    return TrendStable
}

func (p *Predictor) calculateConfidence(historical []models.MetricData, prediction Prediction) float64 {
    // Базовая уверенность
    confidence := 0.8

    // Уменьшаем уверенность на основе волатильности исторических данных
    volatility := p.calculateVolatility(historical)
    confidence *= (1 - volatility)

    // Уменьшаем уверенность с увеличением горизонта прогноза
    timeDiff := prediction.Timestamp.Sub(time.Now())
    confidence *= math.Exp(-float64(timeDiff.Hours()) / 24.0)

    return math.Max(0.1, math.Min(1.0, confidence))
}

func (p *Predictor) calculateVolatility(data []models.MetricData) float64 {
    if len(data) < 2 {
        return 0
    }

    values := make([]float64, len(data))
    for i, d := range data {
        values[i] = d.PowerUsage
    }

    mean, std := stat.MeanStdDev(values, nil)
    return std / mean
}

// Вспомогательные методы для сохранения и загрузки моделей
func (p *Predictor) saveModels() error {
    p.mu.RLock()
    defer p.mu.RUnlock()

    for serverID, model := range p.models {
        data, err := json.Marshal(model)
        if err != nil {
            continue
        }
        
        // Сохраняем модель в файл
        filename := fmt.Sprintf("%s/%s.json", p.config.ModelPath, serverID)
        // Здесь должен быть код для сохранения в файл
    }
    return nil
}

func (p *Predictor) loadModels() error {
    // Здесь должен быть код для загрузки моделей из файлов
    return nil
}

func (p *Predictor) getActiveServers(ctx context.Context) ([]string, error) {
    // Здесь должен быть код для получения списка активных серверов
    return []string{}, nil
}
