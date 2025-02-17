package models

type MetricData struct {
    ServerID      string    `json:"server_id"`
    Timestamp     int64     `json:"timestamp"`
    PowerUsage    float64   `json:"power_usage"`    // Ватты
    CarbonFootprint float64 `json:"carbon_footprint"` // кг CO2
    CPUUsage      float64   `json:"cpu_usage"`      // Процент
    MemoryUsage   float64   `json:"memory_usage"`   // Процент
}

type Server struct {
    ID            string    `json:"id"`
    Provider      string    `json:"provider"` // aws, gcp, azure
    Region        string    `json:"region"`
    InstanceType  string    `json:"instance_type"`
    EcoScore      float64   `json:"eco_score"` // 0-100
}

type Container struct {
    ID            string    `json:"id"`
    ServerID      string    `json:"server_id"`
    ServiceName   string    `json:"service_name"`
    EcoTags       []string  `json:"eco_tags"`
    PowerUsage    float64   `json:"power_usage"`
} 