package ecotags

import (
    "context"
    "sync"
    "time"
    
    "github.com/yourusername/platypus/internal/metrics"
    "github.com/yourusername/platypus/internal/models"
)

// EcoTag представляет экологический тег
type EcoTag struct {
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Score       float64 `json:"score"`        // 0-100
    Weight      float64 `json:"weight"`       // Вес тега при расчете общего рейтинга
    Threshold   float64 `json:"threshold"`    // Пороговое значение для присвоения тега
}

// ServiceEcoProfile содержит экологический профиль сервиса
type ServiceEcoProfile struct {
    ServiceName     string    `json:"service_name"`
    Tags           []string  `json:"tags"`
    EcoScore       float64   `json:"eco_score"`
    PowerUsage     float64   `json:"power_usage"`     // Среднее энергопотребление
    CarbonFootprint float64  `json:"carbon_footprint"` // Углеродный след
    LastUpdate     time.Time `json:"last_update"`
}

type TagManagerConfig struct {
    UpdateInterval time.Duration
    MinDataPoints  int
}

type TagManager struct {
    config     TagManagerConfig
    collector  *metrics.Collector
    analyzer   *metrics.Analyzer
    mu         sync.RWMutex
    profiles   map[string]*ServiceEcoProfile
    tags       map[string]EcoTag
}

func NewTagManager(config TagManagerConfig, collector *metrics.Collector, analyzer *metrics.Analyzer) *TagManager {
    tm := &TagManager{
        config:    config,
        collector: collector,
        analyzer:  analyzer,
        profiles:  make(map[string]*ServiceEcoProfile),
        tags:      make(map[string]EcoTag),
    }
    
    // Инициализация предопределенных тегов
    tm.initializeTags()
    
    return tm
}

func (tm *TagManager) initializeTags() {
    tm.tags = map[string]EcoTag{
        "eco-efficient": {
            Name:        "eco-efficient",
            Description: "Сервис демонстрирует высокую энергоэффективность",
            Score:       100,
            Weight:      1.0,
            Threshold:   80,
        },
        "energy-intensive": {
            Name:        "energy-intensive",
            Description: "Сервис потребляет значительное количество энергии",
            Score:       20,
            Weight:      1.0,
            Threshold:   500, // Ватт
        },
        "carbon-neutral": {
            Name:        "carbon-neutral",
            Description: "Сервис имеет минимальный углеродный след",
            Score:       100,
            Weight:      1.5,
            Threshold:   0.1, // кг CO2
        },
        "optimizable": {
            Name:        "optimizable",
            Description: "Сервис имеет потенциал для оптимизации",
            Score:       50,
            Weight:      0.8,
            Threshold:   60,
        },
        "peak-hours": {
            Name:        "peak-hours",
            Description: "Сервис активен в часы пиковой нагрузки",
            Score:       30,
            Weight:      0.7,
            Threshold:   0.8, // Коэффициент активности в пиковые часы
        },
    }
}

func (tm *TagManager) Start(ctx context.Context) error {
    ticker := time.NewTicker(tm.config.UpdateInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := tm.updateProfiles(ctx); err != nil {
                continue
            }
        }
    }
}

func (tm *TagManager) updateProfiles(ctx context.Context) error {
    containers, err := tm.getActiveContainers(ctx)
    if err != nil {
        return err
    }

    for _, container := range containers {
        profile := tm.analyzeContainer(container)
        if profile != nil {
            tm.mu.Lock()
            tm.profiles[container.ServiceName] = profile
            tm.mu.Unlock()
        }
    }

    return nil
}

func (tm *TagManager) analyzeContainer(container models.Container) *ServiceEcoProfile {
    metrics, err := tm.collector.GetMetrics(container.ServerID)
    if err != nil || len(metrics) < tm.config.MinDataPoints {
        return nil
    }

    // Рассчитываем средние показатели
    var totalPower, totalCarbon float64
    for _, m := range metrics {
        totalPower += m.PowerUsage
        totalCarbon += m.CarbonFootprint
    }
    avgPower := totalPower / float64(len(metrics))
    avgCarbon := totalCarbon / float64(len(metrics))

    // Определяем подходящие теги
    var tags []string
    var totalScore float64
    var totalWeight float64

    for tagName, tag := range tm.tags {
        switch tagName {
        case "eco-efficient":
            if tm.analyzer.CalculateEcoScore(metrics) >= tag.Threshold {
                tags = append(tags, tagName)
                totalScore += tag.Score * tag.Weight
                totalWeight += tag.Weight
            }
        case "energy-intensive":
            if avgPower >= tag.Threshold {
                tags = append(tags, tagName)
                totalScore += tag.Score * tag.Weight
                totalWeight += tag.Weight
            }
        case "carbon-neutral":
            if avgCarbon <= tag.Threshold {
                tags = append(tags, tagName)
                totalScore += tag.Score * tag.Weight
                totalWeight += tag.Weight
            }
        case "optimizable":
            if tm.analyzer.CalculateEcoScore(metrics) < tag.Threshold {
                tags = append(tags, tagName)
                totalScore += tag.Score * tag.Weight
                totalWeight += tag.Weight
            }
        case "peak-hours":
            if tm.isPeakHoursActive(metrics) {
                tags = append(tags, tagName)
                totalScore += tag.Score * tag.Weight
                totalWeight += tag.Weight
            }
        }
    }

    // Рассчитываем итоговый эко-рейтинг
    ecoScore := totalScore / totalWeight
    if totalWeight == 0 {
        ecoScore = 50 // Значение по умолчанию
    }

    return &ServiceEcoProfile{
        ServiceName:     container.ServiceName,
        Tags:           tags,
        EcoScore:       ecoScore,
        PowerUsage:     avgPower,
        CarbonFootprint: avgCarbon,
        LastUpdate:     time.Now(),
    }
}

func (tm *TagManager) GetServiceProfile(serviceName string) (*ServiceEcoProfile, error) {
    tm.mu.RLock()
    defer tm.mu.RUnlock()

    profile, exists := tm.profiles[serviceName]
    if !exists {
        return nil, fmt.Errorf("profile not found for service: %s", serviceName)
    }
    return profile, nil
}

func (tm *TagManager) GetAllProfiles() []*ServiceEcoProfile {
    tm.mu.RLock()
    defer tm.mu.RUnlock()

    profiles := make([]*ServiceEcoProfile, 0, len(tm.profiles))
    for _, profile := range tm.profiles {
        profiles = append(profiles, profile)
    }
    return profiles
}

func (tm *TagManager) isPeakHoursActive(metrics []models.MetricData) bool {
    peakHours := map[int]bool{
        9:  true,
        10: true,
        11: true,
        12: true,
        13: true,
        14: true,
        15: true,
        16: true,
        17: true,
    }

    var peakCount, totalCount int
    for _, m := range metrics {
        hour := time.Unix(m.Timestamp, 0).Hour()
        if peakHours[hour] {
            if m.PowerUsage > 0 {
                peakCount++
            }
            totalCount++
        }
    }

    if totalCount == 0 {
        return false
    }

    return float64(peakCount)/float64(totalCount) >= 0.8
}

func (tm *TagManager) getActiveContainers(ctx context.Context) ([]models.Container, error) {
    // Здесь должна быть реализация получения списка активных контейнеров
    return []models.Container{}, nil
} 