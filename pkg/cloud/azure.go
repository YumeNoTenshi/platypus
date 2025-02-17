package cloud

import (
    "context"
    "time"
    
    "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
    "github.com/Azure/azure-sdk-for-go/services/monitor/mgmt/2021-05-01/insights"
    "../../../platypus/internal/models"
)

type AzureProvider struct {
    vmClient         compute.VirtualMachinesClient
    metricsClient    insights.MetricsClient
    subscriptionID   string
    resourceGroup    string
}

func NewAzureProvider(subscriptionID, resourceGroup string) (*AzureProvider, error) {
    vmClient := compute.NewVirtualMachinesClient(subscriptionID)
    metricsClient := insights.NewMetricsClient(subscriptionID)
    
    // Настройка аутентификации через переменные окружения
    // AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET
    if err := vmClient.Client.Authorizer.Authorize(); err != nil {
        return nil, err
    }

    return &AzureProvider{
        vmClient:       vmClient,
        metricsClient:  metricsClient,
        subscriptionID: subscriptionID,
        resourceGroup:  resourceGroup,
    }, nil
}

func (az *AzureProvider) GetInstances(ctx context.Context) ([]models.Server, error) {
    result, err := az.vmClient.List(ctx, az.resourceGroup)
    if err != nil {
        return nil, err
    }

    var servers []models.Server
    for _, vm := range result.Values() {
        server := models.Server{
            ID:           *vm.ID,
            Provider:     "azure",
            Region:      *vm.Location,
            InstanceType: string(vm.VirtualMachineProperties.HardwareProfile.VMSize),
        }
        servers = append(servers, server)
    }

    return servers, nil
}

func (az *AzureProvider) GetInstanceMetrics(ctx context.Context, instanceID string, period time.Duration) ([]models.MetricData, error) {
    endTime := time.Now()
    startTime := endTime.Add(-period)

    result, err := az.metricsClient.List(ctx, 
        instanceID,
        startTime.Format(time.RFC3339),
        endTime.Format(time.RFC3339),
        "PT5M", // 5-минутные интервалы
        "Percentage CPU",
        "",
        "",
        "Average",
    )
    if err != nil {
        return nil, err
    }

    var metrics []models.MetricData
    for _, metric := range result.Value {
        for _, timeseries := range *metric.Timeseries {
            for _, data := range *timeseries.Data {
                if data.Average != nil {
                    metric := models.MetricData{
                        ServerID:  instanceID,
                        Timestamp: data.TimeStamp.Unix(),
                        CPUUsage:  *data.Average,
                    }
                    metrics = append(metrics, metric)
                }
            }
        }
    }

    return metrics, nil
}
