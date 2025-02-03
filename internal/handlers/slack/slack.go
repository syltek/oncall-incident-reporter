package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/slack-go/slack"
	"github.com/syltek/oncall-incident-reporter/internal/config"
	"github.com/syltek/oncall-incident-reporter/internal/service"
	"github.com/syltek/oncall-incident-reporter/internal/slackmodal"
	apperrors "github.com/syltek/oncall-incident-reporter/pkg/errors"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

// SlackHandler handles Slack-related functionality.
type SlackHandler struct {
	slackService   *service.SlackService
	datadogService *service.DatadogService
	config         *config.Config
}

// NewSlackHandler creates a new SlackHandler instance.
func NewSlackHandler(slackService *service.SlackService, datadogService *service.DatadogService, config *config.Config) *SlackHandler {
	return &SlackHandler{
		slackService:   slackService,
		datadogService: datadogService,
		config:         config,
	}
}

// HandleCommand processes Slack commands to trigger modals.
func (h *SlackHandler) HandleCommand(w http.ResponseWriter, r *http.Request) {
	logutil.Debug("Processing Slack command")
	if err := r.ParseForm(); err != nil {
		h.handleError(w, apperrors.New(http.StatusBadRequest, "Failed to parse form data", apperrors.CategoryClient, err))
		return
	}
	// Debug the form
	logutil.Debug("Form", zap.Any("form", r.Form))

	modal := h.createModal(r.FormValue("trigger_id"))
	if err := modal.SendModal(h.slackService); err != nil {
		h.handleError(w, apperrors.New(http.StatusInternalServerError, "Failed to send modal to Slack", apperrors.CategoryServer, err))
		return
	}

	logutil.Info("Slack modal sent successfully", zap.String("trigger_id", r.FormValue("trigger_id")))
	w.WriteHeader(http.StatusOK)
}

// HandleModalSubmission processes modal submissions from Slack.
func (h *SlackHandler) HandleModalSubmission(w http.ResponseWriter, r *http.Request) {
	logutil.Debug("Processing modal submission")
	if err := r.ParseForm(); err != nil {
		h.handleError(w, apperrors.New(http.StatusBadRequest, "Failed to parse form data", apperrors.CategoryClient, err))
		return
	}

	// Debug the form
	logutil.Debug("Form", zap.Any("form", r.Form))

	modal, err := h.parseModalPayload(r.FormValue("payload"))
	if err != nil {
		h.handleError(w, apperrors.New(http.StatusBadRequest, "Invalid modal payload", apperrors.CategoryClient, err))
		return
	}

	fieldData, err := modal.ParseAllFields()
	if err != nil {
		h.handleError(w, apperrors.New(http.StatusInternalServerError, "Failed to parse modal fields", apperrors.CategoryServer, err))
		return
	}

	username := modal.GetUsername()
	messageText := h.generateIncidentMessage(fieldData, username)

	// Send a message to Slack if the channel is set
	if h.config.SlackConfig != nil && h.config.SlackConfig.ChannelID != "" {
		err = h.sendSlackMessage(messageText)
		if err != nil {
			h.handleError(w, apperrors.New(http.StatusInternalServerError, "Failed to send message to Slack", apperrors.CategoryServer, err))
			return
		}
	}

	// Create a Datadog event
	err = h.createDatadogEvent(messageText, fieldData)
	if err != nil {
		h.handleError(w, apperrors.New(http.StatusInternalServerError, "Failed to create Datadog event", apperrors.CategoryServer, err))
		return
	}

	h.sendResponse(w, map[string]interface{}{"response_action": "clear"})
	logutil.Info("Modal submission processed successfully")
}

// Constants for Datadog event configuration
const (
	eventTitle     = "New on-call alert from slack slash command"
	textBlockStart = "%%% \n"
	textBlockEnd   = "\n %%%"
)

// createDatadogEvent creates a new event in Datadog with the given message
func (h *SlackHandler) createDatadogEvent(messageText string, fieldData map[string]string) error {
	ctx := datadog.NewDefaultContext(context.Background())

	// Enrich the message with event time and event source
	// If local is enabled, use local_execution as the event source
	// Otherwise, use the AWS lambda function name
	event_source := ""
	if h.config.Local.Enabled {
		event_source = "local_execution"
	} else {
		event_source = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	}

	messageText = fmt.Sprintf("%s\nEvents emitted by the %s seen at %s since %s", messageText, event_source, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	eventConfig := h.buildEventConfig(messageText, fieldData)

	ddResponse, err := h.sendEventToDatadog(ctx, eventConfig)
	if err != nil {
		return fmt.Errorf("failed to send event to Datadog: %w", err)
	}

	if err := h.validateResponse(ddResponse); err != nil {
		return err
	}

	logutil.Info("Datadog event created successfully",
		zap.String("url", *ddResponse.GetEvent().Url),
		zap.String("status", ddResponse.GetStatus()))

	return nil
}

// buildEventConfig creates the event configuration
func (h *SlackHandler) buildEventConfig(messageText string, fieldData map[string]string) datadogV1.EventCreateRequest {
	return datadogV1.EventCreateRequest{
		Title:          eventTitle,
		Text:           textBlockStart + messageText + textBlockEnd,
		Priority:       *datadogV1.NewNullableEventPriority(datadogV1.EVENTPRIORITY_NORMAL.Ptr()),
		AlertType:      datadogV1.EVENTALERTTYPE_ERROR.Ptr(),
		Tags:           h.buildEventTags(fieldData),
		SourceTypeName: datadog.PtrString("slack"),
		AggregationKey: datadog.PtrString(h.getAggregationKey()),
	}
}

// buildEventTags returns the tags for the event
func (h *SlackHandler) buildEventTags(fieldData map[string]string) []string {
	return []string{
		"env:" + h.config.Metadata.Environment,
		"team:" + h.config.Metadata.Team,
		"service:" + h.config.Metadata.Service,
		"severity:" + fieldData["input_severity"],
		"domain:" + fieldData["input_domains_affected"],
	}
}

// getAggregationKey returns the aggregation key for the event
func (h *SlackHandler) getAggregationKey() string {
	return h.config.Metadata.Environment + "-" + h.config.Metadata.Service
}

// sendEventToDatadog sends the event to Datadog
func (h *SlackHandler) sendEventToDatadog(ctx context.Context, req datadogV1.EventCreateRequest) (*datadogV1.EventCreateResponse, error) {
	ddResponse, err := h.datadogService.CreateEvent(ctx, req)
	if err != nil {
		logutil.Error("Failed to create Datadog event", zap.Error(err))
		return nil, err
	}

	return ddResponse, nil
}

// validateResponse validates the Datadog API response
func (h *SlackHandler) validateResponse(response *datadogV1.EventCreateResponse) error {
	status := response.GetStatus()

	logutil.Debug("Datadog response status", zap.String("status", status))

	if status != "ok" {
		return fmt.Errorf("failed to create Datadog event: status not expected: %s", status)
	}

	return nil
}

func (h *SlackHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		appErr = apperrors.New(http.StatusInternalServerError, "Internal server error", apperrors.CategoryServer, err)
	}

	logutil.Error(appErr.Message, zap.Error(appErr))
	http.Error(w, appErr.Message, appErr.Code)
}

func (h *SlackHandler) createModal(triggerID string) *slackmodal.Modal {
	logutil.Debug("Creating modal", zap.String("trigger_id", triggerID))
	modal := slackmodal.NewModal(h.config.Modal.Title, triggerID)

	for _, input := range h.config.Modal.Inputs {
		switch input.Type {
		case "select":
			options := h.getSelectOptions(input.Options)
			modal.AddSelectInput(input.Key, input.Label, input.Placeholder, options)
		case "text":
			modal.AddTextInput(input.Key, input.Label, input.Placeholder, false)
		}
	}
	return modal
}

func (h *SlackHandler) getSelectOptions(options []config.Option) []string {
	logutil.Debug("Getting select options", zap.Int("option_count", len(options)))
	result := make([]string, len(options))
	for i, option := range options {
		result[i] = option.Text
	}
	return result
}

func (h *SlackHandler) parseModalPayload(payload string) (*slackmodal.Modal, error) {
	logutil.Debug("Parsing modal payload")
	modal := &slackmodal.Modal{}
	if err := modal.ParsePayload(payload); err != nil {
		return nil, err
	}
	return modal, nil
}

func (h *SlackHandler) generateIncidentMessage(fields map[string]string, username string) string {
	logutil.Debug("Generating incident message",
		zap.String("username", username),
		zap.Any("fields", fields))
	replacements := map[string]string{
		"{{severity}}":         fields["input_severity"],
		"{{domains_affected}}": fields["input_domains_affected"],
		"{{description}}":      fields["input_incident_description"],
		"{{username}}":         username,
	}

	message := h.config.SlackConfig.MessageFormat
	for placeholder, value := range replacements {
		message = strings.ReplaceAll(message, placeholder, value)
	}
	return message
}

func (h *SlackHandler) sendSlackMessage(messageText string) error {
	logutil.Debug("Sending Slack message",
		zap.String("channel_id", h.config.SlackConfig.ChannelID))

	_, _, err := h.slackService.PostMessage(h.config.SlackConfig.ChannelID, slack.MsgOptionText(messageText, false))

	if err != nil {
		return fmt.Errorf("failed to send message to Slack: %w", err)
	}

	return nil
}

func (h *SlackHandler) sendResponse(w http.ResponseWriter, response interface{}) {
	logutil.Debug("Sending response")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.handleError(w, apperrors.New(http.StatusInternalServerError, "Failed to encode response", apperrors.CategoryServer, err))
	}
}
