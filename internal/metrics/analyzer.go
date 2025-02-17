package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"

	"../../../platypus/internal/models"
)

type AnalyzerConfig struct {
	MinDataPoints     int
	SmoothingFactor   float64
	AnomalyThreshold  float64
}

type Analyzer struct {
	config     AnalyzerConfig
	collector  *Collector
}

type MetricAnalysis struct {
	Mean             float64
	Median           float64
	StdDev           float64
	Min              float64
	Max              float64
	Trend            string
	Anomalies        []Anomaly
	PeakUsageTime    time.Time
	EfficiencyScore  float64
}

type Anomaly struct {
	Timestamp time.Time
	Value     float64
	Type      string
	Severity  float64
}

func NewAnalyzer(config AnalyzerConfig, collector *Collector) *Analyzer {
	return &Analyzer{
		config:    config,
		collector: collector,
	}
}

func (a *Analyzer) AnalyzeServerMetrics(serverID string) (*MetricAnalysis, error) {
	metrics, err := a.collector.GetMetrics(serverID)
	if err != nil {
		return nil, err
	}

	if len(metrics) < a.config.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points for analysis")
	}

	analysis := &MetricAnalysis{}
	
	// Базовая статистика
	analysis.Mean = a.calculateMean(metrics)
	analysis.Median = a.calculateMedian(metrics)
	analysis.StdDev = a.calculateStdDev(metrics, analysis.Mean)
	analysis.Min, analysis.Max = a.calculateMinMax(metrics)
	
	// Анализ тренда
	analysis.Trend = a.analyzeTrend(metrics)
	
	// Поиск аномалий
	analysis.Anomalies = a.detectAnomalies(metrics, analysis.Mean, analysis.StdDev)
	
	// Определение пикового времени использования
	analysis.PeakUsageTime = a.findPeakUsageTime(metrics)
	
	// Расчет общего показателя эффективности
	analysis.EfficiencyScore = a.calculateEfficiencyScore(metrics)
	
	return analysis, nil
}

func (a *Analyzer) calculateMean(metrics []models.MetricData) float64 {
	var sum float64
	for _, m := range metrics {
		sum += m.PowerUsage
	}
	return sum / float64(len(metrics))
}

func (a *Analyzer) calculateMedian(metrics []models.MetricData) float64 {
	values := make([]float64, len(metrics))
	for i, m := range metrics {
		values[i] = m.PowerUsage
	}
	sort.Float64s(values)
	
	mid := len(values) / 2
	if len(values)%2 == 0 {
		return (values[mid-1] + values[mid]) / 2
	}
	return values[mid]
}

func (a *Analyzer) calculateStdDev(metrics []models.MetricData, mean float64) float64 {
	var sumSquares float64
	for _, m := range metrics {
		diff := m.PowerUsage - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(metrics)))
}

func (a *Analyzer) calculateMinMax(metrics []models.MetricData) (float64, float64) {
	min := metrics[0].PowerUsage
	max := metrics[0].PowerUsage
	
	for _, m := range metrics {
		if m.PowerUsage < min {
			min = m.PowerUsage
		}
		if m.PowerUsage > max {
			max = m.PowerUsage
		}
	}
	
	return min, max
}

func (a *Analyzer) analyzeTrend(metrics []models.MetricData) string {
	if len(metrics) < 2 {
		return "stable"
	}

	// Простой линейный тренд
	firstHalf := metrics[:len(metrics)/2]
	secondHalf := metrics[len(metrics)/2:]
	
	firstMean := a.calculateMean(firstHalf)
	secondMean := a.calculateMean(secondHalf)
	
	diff := secondMean - firstMean
	threshold := 0.1 * firstMean
	
	if diff > threshold {
		return "increasing"
	} else if diff < -threshold {
		return "decreasing"
	}
	return "stable"
}

func (a *Analyzer) detectAnomalies(metrics []models.MetricData, mean, stdDev float64) []Anomaly {
	var anomalies []Anomaly
	
	for _, m := range metrics {
		zScore := math.Abs(m.PowerUsage - mean) / stdDev
		if zScore > a.config.AnomalyThreshold {
			anomaly := Anomaly{
				Timestamp: time.Unix(m.Timestamp, 0),
				Value:     m.PowerUsage,
				Type:      a.classifyAnomaly(m.PowerUsage, mean),
				Severity:  zScore,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

func (a *Analyzer) classifyAnomaly(value, mean float64) string {
	if value > mean {
		return "spike"
	}
	return "drop"
}

func (a *Analyzer) findPeakUsageTime(metrics []models.MetricData) time.Time {
	var maxUsage float64
	var peakTime time.Time
	
	for _, m := range metrics {
		if m.PowerUsage > maxUsage {
			maxUsage = m.PowerUsage
			peakTime = time.Unix(m.Timestamp, 0)
		}
	}
	
	return peakTime
}

func (a *Analyzer) calculateEfficiencyScore(metrics []models.MetricData) float64 {
	if len(metrics) == 0 {
		return 0
	}

	// Базовый показатель на основе энергопотребления
	powerScore := a.calculatePowerScore(metrics)
	
	// Учитываем утилизацию CPU
	utilizationScore := a.calculateUtilizationScore(metrics)
	
	// Учитываем углеродный след
	carbonScore := a.calculateCarbonScore(metrics)
	
	// Взвешенная сумма всех показателей
	return (powerScore*0.4 + utilizationScore*0.3 + carbonScore*0.3) * 100
}

func (a *Analyzer) calculatePowerScore(metrics []models.MetricData) float64 {
	mean := a.calculateMean(metrics)
	// Нормализация: чем меньше энергопотребление, тем выше счет
	return math.Max(0, 1-mean/1000) // 1000W как базовое значение
}

func (a *Analyzer) calculateUtilizationScore(metrics []models.MetricData) float64 {
	var totalUtil float64
	for _, m := range metrics {
		// Оптимальная утилизация около 70%
		score := 1 - math.Abs(0.7-m.CPUUsage/100)
		totalUtil += score
	}
	return totalUtil / float64(len(metrics))
}

func (a *Analyzer) calculateCarbonScore(metrics []models.MetricData) float64 {
	var totalCarbon float64
	for _, m := range metrics {
		totalCarbon += m.CarbonFootprint
	}
	avgCarbon := totalCarbon / float64(len(metrics))
	// Нормализация: чем меньше углеродный след, тем выше счет
	return math.Max(0, 1-avgCarbon)
} 