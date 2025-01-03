package datadogmonitors

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

type MonitorService struct {
	monitorsApi *datadogV1.MonitorsApi
}

func NewMonitorService() *MonitorService {
	configuration := datadog.NewConfiguration()
	host := os.Getenv("DD_HOST")

	if host == "" {
		host = "https://api.datadoghq.eu"
	}
	// Set Datadog API host
	configuration.Servers = datadog.ServerConfigurations{
		{
			URL: host,
		},
	}

	apiClient := datadog.NewAPIClient(configuration)

	return &MonitorService{
		monitorsApi: datadogV1.NewMonitorsApi(apiClient),
	}
}

func addAPIKeysToContext(ctx context.Context) (context.Context, error) {
	// Check that environment variables are set
	apiKey := os.Getenv("DD_CLIENT_API_KEY")
	appKey := os.Getenv("DD_CLIENT_APP_KEY")
	if apiKey == "" || appKey == "" {
		return nil, fmt.Errorf("DD_CLIENT_API_KEY and DD_CLIENT_APP_KEY must be set")
	}

	// Add API key and app key to the context
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {Key: apiKey},
			"appKeyAuth": {Key: appKey},
		},
	)
	return ctx, nil
}

func (s *MonitorService) GetMonitorQuery(ctx context.Context, monitorID int64) (string, error) {
    ctx, err := addAPIKeysToContext(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to add API keys to context: %w", err)
    }

    monitor, _, err := s.monitorsApi.GetMonitor(ctx, monitorID)
    if err != nil {
        return "", fmt.Errorf("failed to get monitor %d: %w", monitorID, err)
    }
    return monitor.Query, nil
}

func (s *MonitorService) updateMonitor(ctx context.Context, monitorID int64, monitorUpdateRequest datadogV1.MonitorUpdateRequest) error {
    ctx, err := addAPIKeysToContext(ctx)
    if err != nil {
        return fmt.Errorf("failed to add API keys to context: %w", err)
    }

    _, _, err = s.monitorsApi.UpdateMonitor(ctx, monitorID, monitorUpdateRequest)
    if err != nil {
        return fmt.Errorf("failed to update monitor %d: %w", monitorID, err)
    }
    return nil
}

func (s *MonitorService) UpdateMonitorDescription(ctx context.Context, monitorID int64, description string) error {
	ctx, err := addAPIKeysToContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to add API keys to context: %w", err)
	}

	monitorUpdateRequest := datadogV1.MonitorUpdateRequest{
		Message: &description,
	}
	return s.updateMonitor(ctx, monitorID, monitorUpdateRequest)
}

func (s *MonitorService) UpdateMonitorQuery(ctx context.Context, monitorID int64, query string) error {
    ctx, err := addAPIKeysToContext(ctx)
    if err != nil {
        return fmt.Errorf("failed to add API keys to context: %w", err)
    }

    monitorUpdateRequest := datadogV1.MonitorUpdateRequest{
        Query: &query,
    }
    return s.updateMonitor(ctx, monitorID, monitorUpdateRequest)
}

func (s *MonitorService) UpdateMonitorCriticalThreshold(ctx context.Context, monitorID int64, criticalThreshold float64) error {
	ctx, err := addAPIKeysToContext(ctx)
	if err != nil {
		return err
	}

	monitorUpdateRequest := datadogV1.MonitorUpdateRequest{
		Options: &datadogV1.MonitorOptions{
			Thresholds: &datadogV1.MonitorThresholds{
				Critical: &criticalThreshold,
			},
		},
	}
	return s.updateMonitor(ctx, monitorID, monitorUpdateRequest)
}

// Add new helper method
func (ms *MonitorService) GetMonitorNewQuery(query string, newThreshold float64) (string, error) {
	re := regexp.MustCompile(`[><]=?\s*([\d.]+)`)
	updatedQuery := re.ReplaceAllStringFunc(query, func(match string) string {
		return fmt.Sprintf("%s %f", match[:1], newThreshold)
	})
	return updatedQuery, nil
}
