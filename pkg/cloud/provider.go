package cloud

import (
    "context"
    "time"
    
    "github.com/YumeNoTenshi/platypus/internal/models"
)

// CloudProvider определяет интерфейс для работы с облачными провайдерами
type CloudProvider interface {
    // GetInstances возвращает список всех инстансов
    GetInstances(ctx context.Context) ([]models.Server, error)
    
    // GetInstanceMetrics возвращает метрики для конкретного инстанса
    GetInstanceMetrics(ctx context.Context, instanceID string, period time.Duration) ([]models.MetricData, error)
    
    // MigrateContainer перемещает контейнер на другой инстанс
    MigrateContainer(ctx context.Context, containerID, sourceID, targetID string) error
    
    // GetPowerUsage возвращает данные об энергопотреблении
    GetPowerUsage(ctx context.Context, instanceID string) (float64, error)
} 