package scaling

import (
    "context"
    "sync"
    "time"
    
    "github.com/yourusername/platypus/internal/metrics"
    "github.com/yourusername/platypus/internal/models"
    "github.com/yourusername/platypus/pkg/cloud"
)

type AutoscalerConfig struct {
    CPUThresholdHigh    float64       // Верхний порог CPU для масштабирования (например, 80%)
    CPUThresholdLow     float64       // Нижний порог CPU для уменьшения (например, 20%)
    PowerThresholdHigh  float64       // Верхний порог энергопотребления (Вт)
    ScaleUpCooldown     time.Duration // Период ожидания между масштабированиями вверх
    ScaleDownCooldown   time.Duration // Период ожидания между масштабированиями вниз
    EvaluationInterval  time.Duration // Интервал проверки метрик
}

type Autoscaler struct {
    config      AutoscalerConfig
    collector   *metrics.Collector
    analyzer    *metrics.Analyzer
    provider    cloud.CloudProvider
    mu          sync.RWMutex
    lastScaleUp time.Time
    lastScaleDown time.Time
}

func NewAutoscaler(config AutoscalerConfig, collector *metrics.Collector, analyzer *metrics.Analyzer, provider cloud.CloudProvider) *Autoscaler {
    return &Autoscaler{
        config:    config,
        collector: collector,
        analyzer:  analyzer,
        provider:  provider,
    }
}

func (a *Autoscaler) Start(ctx context.Context) error {
    ticker := time.NewTicker(a.config.EvaluationInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := a.evaluate(ctx); err != nil {
                // Логируем ошибку, но продолжаем работу
                continue
            }
        }
    }
}

func (a *Autoscaler) evaluate(ctx context.Context) error {
    // Получаем список всех серверов
    servers, err := a.provider.GetInstances(ctx)
    if err != nil {
        return err
    }

    for _, server := range servers {
        metrics, err := a.collector.GetMetrics(server.ID)
        if err != nil {
            continue
        }

        if len(metrics) == 0 {
            continue
        }

        // Анализируем последние метрики
        lastMetric := metrics[len(metrics)-1]
        
        // Проверяем необходимость масштабирования
        if a.shouldScaleUp(lastMetric) {
            if err := a.scaleUp(ctx, server); err != nil {
                return err
            }
        } else if a.shouldScaleDown(lastMetric) {
            if err := a.scaleDown(ctx, server); err != nil {
                return err
            }
        }
    }

    return nil
}

func (a *Autoscaler) shouldScaleUp(metric models.MetricData) bool {
    a.mu.RLock()
    defer a.mu.RUnlock()

    // Проверяем, прошло ли достаточно времени с последнего масштабирования
    if time.Since(a.lastScaleUp) < a.config.ScaleUpCooldown {
        return false
    }

    // Проверяем пороги
    return metric.CPUUsage > a.config.CPUThresholdHigh ||
           metric.PowerUsage > a.config.PowerThresholdHigh
}

func (a *Autoscaler) shouldScaleDown(metric models.MetricData) bool {
    a.mu.RLock()
    defer a.mu.RUnlock()

    if time.Since(a.lastScaleDown) < a.config.ScaleDownCooldown {
        return false
    }

    return metric.CPUUsage < a.config.CPUThresholdLow
}

func (a *Autoscaler) scaleUp(ctx context.Context, server models.Server) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    // Находим сервер с наименьшим энергопотреблением для миграции
    targetServer, err := a.findEnergyEfficientServer(ctx)
    if err != nil {
        return err
    }

    // Получаем список контейнеров на сервере
    containers, err := a.getServerContainers(ctx, server.ID)
    if err != nil {
        return err
    }

    // Мигрируем контейнеры на новый сервер
    for _, container := range containers {
        if err := a.provider.MigrateContainer(ctx, container.ID, server.ID, targetServer.ID); err != nil {
            continue
        }
    }

    a.lastScaleUp = time.Now()
    return nil
}

func (a *Autoscaler) scaleDown(ctx context.Context, server models.Server) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    // Проверяем эко-рейтинг сервера
    ecoScore := a.analyzer.CalculateEcoScore([]models.MetricData{})
    if ecoScore > 80 {
        // Если сервер энергоэффективен, сохраняем его
        return nil
    }

    // Находим более энергоэффективный сервер для миграции
    targetServer, err := a.findEnergyEfficientServer(ctx)
    if err != nil {
        return err
    }

    // Мигрируем все контейнеры
    containers, err := a.getServerContainers(ctx, server.ID)
    if err != nil {
        return err
    }

    for _, container := range containers {
        if err := a.provider.MigrateContainer(ctx, container.ID, server.ID, targetServer.ID); err != nil {
            return err
        }
    }

    a.lastScaleDown = time.Now()
    return nil
}

func (a *Autoscaler) findEnergyEfficientServer(ctx context.Context) (models.Server, error) {
    servers, err := a.provider.GetInstances(ctx)
    if err != nil {
        return models.Server{}, err
    }

    var bestServer models.Server
    var bestScore float64

    for _, server := range servers {
        metrics, err := a.collector.GetMetrics(server.ID)
        if err != nil {
            continue
        }

        score := a.analyzer.CalculateEcoScore(metrics)
        if score > bestScore {
            bestScore = score
            bestServer = server
        }
    }

    return bestServer, nil
}

func (a *Autoscaler) getServerContainers(ctx context.Context, serverID string) ([]models.Container, error) {
    // Здесь должна быть реализация получения списка контейнеров с сервера
    // Можно использовать Kubernetes API или другие механизмы
    return []models.Container{}, nil
} 