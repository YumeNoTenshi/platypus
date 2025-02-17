package cloud

import (
	"context"
	"fmt"
	"time"

	compute "google.golang.org/api/compute/v1"
	monitoring "google.golang.org/api/monitoring/v3"
	"../../../platypus/internal/models"
)

type GCPProvider struct {
	computeService    *compute.Service
	monitoringService *monitoring.Service
	projectID        string
	zone             string
}

func NewGCPProvider(ctx context.Context, projectID, zone string) (*GCPProvider, error) {
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	monitoringService, err := monitoring.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return &GCPProvider{
		computeService:    computeService,
		monitoringService: monitoringService,
		projectID:        projectID,
		zone:             zone,
	}, nil
}

func (g *GCPProvider) GetInstances(ctx context.Context) ([]models.Server, error) {
	instances, err := g.computeService.Instances.List(g.projectID, g.zone).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	var servers []models.Server
	for _, instance := range instances.Items {
		server := models.Server{
			ID:           instance.Id,
			Provider:     "gcp",
			Region:      g.zone,
			InstanceType: instance.MachineType,
		}
		servers = append(servers, server)
	}

	return servers, nil
}

func (g *GCPProvider) GetInstanceMetrics(ctx context.Context, instanceID string, period time.Duration) ([]models.MetricData, error) {
	endTime := time.Now()
	startTime := endTime.Add(-period)

	request := &monitoring.ListTimeSeriesRequest{
		Filter: fmt.Sprintf(
			`metric.type="compute.googleapis.com/instance/cpu/utilization" AND 
			 resource.labels.instance_id="%s"`,
			instanceID,
		),
		Interval: &monitoring.TimeInterval{
			StartTime: startTime.Format(time.RFC3339),
			EndTime:   endTime.Format(time.RFC3339),
		},
	}

	resp, err := g.monitoringService.Projects.TimeSeries.List("projects/"+g.projectID).
		Filter(request.Filter).
		IntervalStartTime(request.Interval.StartTime).
		IntervalEndTime(request.Interval.EndTime).
		Do()
	if err != nil {
		return nil, err
	}

	var metrics []models.MetricData
	for _, series := range resp.TimeSeries {
		for _, point := range series.Points {
			metric := models.MetricData{
				ServerID:  instanceID,
				Timestamp: point.Interval.EndTime,
				CPUUsage:  point.Value.DoubleValue,
			}
			metrics = append(metrics, metric)
		}
	}

	return metrics, nil
}
