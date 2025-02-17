package migration

import (
    "context"
    "sort"
    "sync"
    "time"
    
    "github.com/YumeNoTenshi/platypus/internal/metrics"
    "github.com/YumeNoTenshi/platypus/internal/models"
    "github.com/YumeNoTenshi/platypus/pkg/cloud"
)

type MigrationPlan struct {
    ContainerID     string
    SourceServerID  string
    TargetServerID  string
    Priority        int     // 1-10, где 10 - наивысший приоритет
    PowerSaving     float64 // Ожидаемая экономия энергии в ваттах
    DowntimeEstimate time.Duration
}

type PlannerConfig struct {
    MinPowerSaving      float64       // Минимальная экономия энергии для миграции (ватты)
    MaxDowntime         time.Duration // Максимальное допустимое время простоя
    PlanningInterval    time.Duration // Интервал планирования миграций
    ConcurrentMigrations int         // Максимальное количество одновременных миграций
}

type Planner struct {
    config      PlannerConfig
    collector   *metrics.Collector
    analyzer    *metrics.Analyzer
    provider    cloud.CloudProvider
    mu          sync.RWMutex
    activePlans map[string]*MigrationPlan // ContainerID -> Plan
}

func NewPlanner(config PlannerConfig, collector *metrics.Collector, analyzer *metrics.Analyzer, provider cloud.CloudProvider) *Planner {
    return &Planner{
        config:      config,
        collector:   collector,
        analyzer:    analyzer,
        provider:    provider,
        activePlans: make(map[string]*MigrationPlan),
    }
}

func (p *Planner) Start(ctx context.Context) error {
    ticker := time.NewTicker(p.config.PlanningInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := p.planMigrations(ctx); err != nil {
                // Логируем ошибку, но продолжаем работу
                continue
            }
            if err := p.executeMigrations(ctx); err != nil {
                continue
            }
        }
    }
}

func (p *Planner) planMigrations(ctx context.Context) error {
    // Получаем все серверы
    servers, err := p.provider.GetInstances(ctx)
    if err != nil {
        return err
    }

    // Сортируем серверы по энергоэффективности
    sort.Slice(servers, func(i, j int) bool {
        scoreI := p.getServerEcoScore(servers[i].ID)
        scoreJ := p.getServerEcoScore(servers[j].ID)
        return scoreI > scoreJ
    })

    // Анализируем каждый сервер с низкой энергоэффективностью
    for _, sourceServer := range servers {
        if p.getServerEcoScore(sourceServer.ID) > 70 {
            continue // Сервер достаточно эффективен
        }

        // Получаем контейнеры на сервере
        containers, err := p.getServerContainers(ctx, sourceServer.ID)
        if err != nil {
            continue
        }

        // Для каждого контейнера ищем лучший целевой сервер
        for _, container := range containers {
            if _, exists := p.activePlans[container.ID]; exists {
                continue // Для этого контейнера уже есть план миграции
            }

            bestPlan := p.findBestMigrationPlan(ctx, container, sourceServer, servers)
            if bestPlan != nil {
                p.mu.Lock()
                p.activePlans[container.ID] = bestPlan
                p.mu.Unlock()
            }
        }
    }

    return nil
}

func (p *Planner) findBestMigrationPlan(
    ctx context.Context,
    container models.Container,
    sourceServer models.Server,
    targetServers []models.Server,
) *MigrationPlan {
    var bestPlan *MigrationPlan
    var maxPowerSaving float64

    for _, targetServer := range targetServers {
        if targetServer.ID == sourceServer.ID {
            continue
        }

        // Оцениваем потенциальную экономию энергии
        powerSaving := p.estimatePowerSaving(container, sourceServer, targetServer)
        if powerSaving < p.config.MinPowerSaving {
            continue
        }

        // Оцениваем время простоя при миграции
        downtime := p.estimateDowntime(container, sourceServer, targetServer)
        if downtime > p.config.MaxDowntime {
            continue
        }

        // Если это лучший вариант - сохраняем
        if powerSaving > maxPowerSaving {
            maxPowerSaving = powerSaving
            bestPlan = &MigrationPlan{
                ContainerID:     container.ID,
                SourceServerID:  sourceServer.ID,
                TargetServerID:  targetServer.ID,
                Priority:        p.calculatePriority(powerSaving, downtime),
                PowerSaving:     powerSaving,
                DowntimeEstimate: downtime,
            }
        }
    }

    return bestPlan
}

func (p *Planner) executeMigrations(ctx context.Context) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Сортируем планы по приоритету
    var plans []*MigrationPlan
    for _, plan := range p.activePlans {
        plans = append(plans, plan)
    }
    sort.Slice(plans, func(i, j int) bool {
        return plans[i].Priority > plans[j].Priority
    })

    // Выполняем миграции с учетом ограничения на количество одновременных операций
    var wg sync.WaitGroup
    sem := make(chan struct{}, p.config.ConcurrentMigrations)

    for _, plan := range plans {
        wg.Add(1)
        sem <- struct{}{} // Захватываем слот для миграции

        go func(plan *MigrationPlan) {
            defer wg.Done()
            defer func() { <-sem }() // Освобождаем слот

            err := p.provider.MigrateContainer(
                ctx,
                plan.ContainerID,
                plan.SourceServerID,
                plan.TargetServerID,
            )
            if err == nil {
                p.mu.Lock()
                delete(p.activePlans, plan.ContainerID)
                p.mu.Unlock()
            }
        }(plan)
    }

    wg.Wait()
    return nil
}

func (p *Planner) getServerEcoScore(serverID string) float64 {
    metrics, err := p.collector.GetMetrics(serverID)
    if err != nil {
        return 0
    }
    return p.analyzer.CalculateEcoScore(metrics)
}

func (p *Planner) estimatePowerSaving(
    container models.Container,
    sourceServer, targetServer models.Server,
) float64 {
    sourcePower := container.PowerUsage
    // Оценка энергопотребления на целевом сервере
    targetPower := sourcePower * (p.getServerEcoScore(targetServer.ID) / 
                                 p.getServerEcoScore(sourceServer.ID))
    return sourcePower - targetPower
}

func (p *Planner) estimateDowntime(
    container models.Container,
    sourceServer, targetServer models.Server,
) time.Duration {
    // Базовое время миграции
    baseTime := 30 * time.Second

    // Учитываем расстояние между регионами
    if sourceServer.Region != targetServer.Region {
        baseTime += 1 * time.Minute
    }

    return baseTime
}

func (p *Planner) calculatePriority(powerSaving float64, downtime time.Duration) int {
    // Приоритет зависит от экономии энергии и времени простоя
    priority := int((powerSaving / p.config.MinPowerSaving) * 10)
    
    // Уменьшаем приоритет, если время простоя большое
    if downtime > p.config.MaxDowntime/2 {
        priority -= 2
    }

    // Ограничиваем приоритет диапазоном 1-10
    if priority < 1 {
        priority = 1
    }
    if priority > 10 {
        priority = 10
    }

    return priority
}

func (p *Planner) getServerContainers(ctx context.Context, serverID string) ([]models.Container, error) {
    // Здесь должна быть реализация получения списка контейнеров с сервера
    // Можно использовать Kubernetes API или другие механизмы
    return []models.Container{}, nil
} 