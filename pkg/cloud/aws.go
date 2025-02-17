package cloud

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"../../../platypus/internal/models"
)

type AWSProvider struct {
	ec2Client        *ec2.Client
	cloudWatchClient *cloudwatch.Client
	region          string
}

func NewAWSProvider(region string) (*AWSProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	return &AWSProvider{
		ec2Client:        ec2.NewFromConfig(cfg),
		cloudWatchClient: cloudwatch.NewFromConfig(cfg),
		region:          region,
	}, nil
}

func (a *AWSProvider) GetInstances(ctx context.Context) ([]models.Server, error) {
	input := &ec2.DescribeInstancesInput{}
	result, err := a.ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, err
	}

	var servers []models.Server
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			server := models.Server{
				ID:           *instance.InstanceId,
				Provider:     "aws",
				Region:      a.region,
				InstanceType: string(instance.InstanceType),
			}
			servers = append(servers, server)
		}
	}

	return servers, nil
}

func (a *AWSProvider) GetInstanceMetrics(ctx context.Context, instanceID string, period time.Duration) ([]models.MetricData, error) {
	endTime := time.Now()
	startTime := endTime.Add(-period)

	input := &cloudwatch.GetMetricDataInput{
		StartTime: &startTime,
		EndTime:   &endTime,
		MetricDataQueries: []cloudwatch.MetricDataQuery{
			{
				Id: aws.String("cpu"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("AWS/EC2"),
						MetricName: aws.String("CPUUtilization"),
						Dimensions: []cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instanceID),
							},
						},
					},
					Period: aws.Int32(300),
					Stat:   aws.String("Average"),
				},
			},
		},
	}

	result, err := a.cloudWatchClient.GetMetricData(ctx, input)
	if err != nil {
		return nil, err
	}

	var metrics []models.MetricData
	for i, timestamp := range result.MetricDataResults[0].Timestamps {
		metric := models.MetricData{
			ServerID:  instanceID,
			Timestamp: timestamp.Unix(),
			CPUUsage:  *result.MetricDataResults[0].Values[i],
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (a *AWSProvider) GetPowerUsage(ctx context.Context, instanceID string) (float64, error) {
	// В AWS нет прямого API для получения энергопотребления
	// Используем приблизительные расчеты на основе типа инстанса и его загрузки
	instance, err := a.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return 0, err
	}

	// Примерный расчет энергопотребления на основе типа инстанса
	instanceType := instance.Reservations[0].Instances[0].InstanceType
	return calculatePowerUsage(string(instanceType)), nil
}
