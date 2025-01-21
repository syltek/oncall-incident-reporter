package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

type IDatadogEventsAPI interface {
	CreateEvent(ctx context.Context, body datadogV1.EventCreateRequest) (datadogV1.EventCreateResponse, *http.Response, error)
}

type DatadogService struct {
	client IDatadogEventsAPI
}

func NewDatadogService(client IDatadogEventsAPI) *DatadogService {
	return &DatadogService{client: client}
}

func (c *DatadogService) CreateEvent(ctx context.Context, event datadogV1.EventCreateRequest) (*datadogV1.EventCreateResponse, error) {
	resp, _, err := c.client.CreateEvent(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}
	return &resp, nil
}
