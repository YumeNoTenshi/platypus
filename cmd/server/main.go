package main

import (
    "context"
    "log"
    "net/http"
    "time"
    
    "../../../platypus/internal/api"
    "../../../internal/metrics"
    "../../../internal/scaling"
    "../../../internal/migration"
    "../../../pkg/cloud"
    "../../../pkg/ml"
    "../../../internal/ecotags"
)

func main() {
    // Инициализация коллектора метрик
    collectorConfig := metrics.CollectorConfig{
        RetentionPeriod:    168 * time.Hour,
        CollectionInterval: time.Minute,
        BatchSize:         100,
        BufferSize:        1000,
    }

    collector := metrics.NewCollector(collectorConfig)
    
    // Инициализация анализатора
    analyzerConfig := metrics.AnalyzerConfig{
        MinDataPoints:    10,
        SmoothingFactor:  0.2,
        AnomalyThreshold: 2.5,
    }

    analyzer := metrics.NewAnalyzer(analyzerConfig, collector)
    
    // Инициализация HTTP сервера
    server := api.NewServer(collector, analyzer)
    
    config := scaling.AutoscalerConfig{
        CPUThresholdHigh:    80.0,
        CPUThresholdLow:     20.0,
        PowerThresholdHigh:  1000.0,
        ScaleUpCooldown:     5 * time.Minute,
        ScaleDownCooldown:   15 * time.Minute,
        EvaluationInterval:  1 * time.Minute,
    }

    autoscaler := scaling.NewAutoscaler(config, collector, analyzer, cloud.NewCloudProvider())
    go autoscaler.Start(context.Background())

    plannerConfig := migration.PlannerConfig{
        MinPowerSaving:      100.0,
        MaxDowntime:         2 * time.Minute,
        PlanningInterval:    5 * time.Minute,
        ConcurrentMigrations: 3,
    }

    planner := migration.NewPlanner(plannerConfig, collector, analyzer, cloud.NewCloudProvider())
    go planner.Start(context.Background())

    predictorConfig := ml.PredictorConfig{
        HistoryWindow:    168 * time.Hour,
        PredictionWindow: 24 * time.Hour,
        UpdateInterval:   1 * time.Hour,
        MinDataPoints:    24,
        ModelPath:        "./data/models",
    }

    predictor := ml.NewPredictor(predictorConfig, collector)
    go predictor.Start(context.Background())

    tagManagerConfig := ecotags.TagManagerConfig{
        UpdateInterval: 15 * time.Minute,
        MinDataPoints:  10,
    }

    tagManager := ecotags.NewTagManager(tagManagerConfig, collector, analyzer)
    go tagManager.Start(context.Background())

    go collector.Start(context.Background())

    log.Println("Запуск Platypus сервера на порту :8080")
    if err := http.ListenAndServe(":8080", server.Router()); err != nil {
        log.Fatal(err)
    }
} 