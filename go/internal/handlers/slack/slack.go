package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
	"github.com/syltek/oncall-incident-reporter/internal/config"
	"github.com/syltek/oncall-incident-reporter/internal/slackmodal"
	"github.com/syltek/oncall-incident-reporter/pkg/datadogmonitors"
	"github.com/syltek/oncall-incident-reporter/pkg/logutil"
	"go.uber.org/zap"
)

// SlackHandler handles Slack-related functionality.
type SlackHandler struct {
	cfg         *config.Config
	slackClient *slack.Client
}

// NewSlackHandler creates a new SlackHandler instance.
func NewSlackHandler(cfg *config.Config) *SlackHandler {
	return &SlackHandler{
		cfg:         cfg,
		slackClient: slack.New(cfg.SlackConfig.Token),
	}
}

// HandleCommand processes Slack commands to trigger modals.
func (h *SlackHandler) HandleCommand(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.handleError(w, "Failed to parse form data", http.StatusBadRequest, err)
		return
	}

	modal := h.createModal(r.FormValue("trigger_id"))
	if err := modal.SendModal(h.slackClient); err != nil {
		h.handleError(w, "Failed to send modal to Slack", http.StatusInternalServerError, err)
		return
	}

	logutil.Info("Slack modal sent successfully", zap.String("trigger_id", r.FormValue("trigger_id")))
	w.WriteHeader(http.StatusOK)
}

// HandleModalSubmission processes modal submissions from Slack.
func (h *SlackHandler) HandleModalSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.handleError(w, "Failed to parse form data", http.StatusBadRequest, err)
		return
	}

	modal, err := h.parseModalPayload(r.FormValue("payload"))
	if err != nil {
		h.handleError(w, "Invalid modal payload", http.StatusBadRequest, err)
		return
	}

	fieldData, err := modal.ParseAllFields()
	if err != nil {
		h.handleError(w, "Failed to parse modal fields", http.StatusInternalServerError, err)
		return
	}

	username := modal.GetUsername()
	messageText := h.generateIncidentMessage(fieldData, username)

	h.sendSlackMessage(messageText, w)
	h.triggerHighSeverityMonitor(fieldData, messageText, w)

	h.sendResponse(w, map[string]interface{}{"response_action": "clear"})
	logutil.Info("Modal submission processed successfully")
}

func (h *SlackHandler) handleError(w http.ResponseWriter, message string, statusCode int, err error) {
	logutil.Error(message, zap.Error(err))
	http.Error(w, message, statusCode)
}

func (h *SlackHandler) createModal(triggerID string) *slackmodal.Modal {
	modal := slackmodal.NewModal(h.cfg.Modal.Title, triggerID)

	for _, input := range h.cfg.Modal.Inputs {
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
	result := make([]string, len(options))
	for i, option := range options {
		result[i] = option.Text
	}
	return result
}

func (h *SlackHandler) parseModalPayload(payload string) (*slackmodal.Modal, error) {
	modal := &slackmodal.Modal{}
	if err := modal.ParsePayload(payload); err != nil {
		return nil, err
	}
	return modal, nil
}

func (h *SlackHandler) generateIncidentMessage(fields map[string]string, username string) string {
	replacements := map[string]string{
		"{{severity}}":         fields["input_severity"],
		"{{domains_affected}}": fields["input_domains_affected"],
		"{{description}}":      fields["input_incident_description"],
		"{{username}}":         username,
	}

	message := h.cfg.SlackConfig.MessageFormat
	for placeholder, value := range replacements {
		message = strings.ReplaceAll(message, placeholder, value)
	}
	return message
}

func (h *SlackHandler) sendSlackMessage(messageText string, w http.ResponseWriter) {
	if h.cfg.SlackConfig == nil || h.cfg.SlackConfig.ChannelID == "" {
		return
	}

	_, _, err := h.slackClient.PostMessage(h.cfg.SlackConfig.ChannelID, slack.MsgOptionText(messageText, false))
	if err != nil {
		h.handleError(w, "Failed to send message to Slack", http.StatusInternalServerError, err)
	}
}

func (h *SlackHandler) triggerHighSeverityMonitor(fields map[string]string, messageText string, w http.ResponseWriter) {
	if h.cfg.MonitorConfig == nil || fields["input_severity"] != "High" {
		return
	}

	if err := h.triggerMonitor(messageText, h.cfg.MonitorConfig.AlertRecipient); err != nil {
		h.handleError(w, "Failed to trigger monitor", http.StatusInternalServerError, err)
	}
}

func (h *SlackHandler) sendResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.handleError(w, "Failed to encode response", http.StatusInternalServerError, err)
	}
}

func (h *SlackHandler) triggerMonitor(description, alertRecipient string) error {
	monitorService := datadogmonitors.NewMonitorService()
	ctx := context.Background()

	description = fmt.Sprintf("%s\n%s", description, alertRecipient)

	if err := h.updateMonitor(*monitorService, ctx, description); err != nil {
		return err
	}

	return nil
}

func (h *SlackHandler) updateMonitor(monitorService datadogmonitors.MonitorService, ctx context.Context, description string) error {
	if err := monitorService.UpdateMonitorDescription(ctx, h.cfg.MonitorConfig.ID, description); err != nil {
		return fmt.Errorf("failed to update monitor description: %w", err)
	}

	monitorQuery, err := monitorService.GetMonitorQuery(ctx, h.cfg.MonitorConfig.ID)
	if err != nil {
		return fmt.Errorf("failed to get monitor query: %w", err)
	}

	newQuery, err := monitorService.GetMonitorNewQuery(monitorQuery, h.cfg.MonitorConfig.NewCriticalThreshold)
	if err != nil {
		return fmt.Errorf("failed to generate new query: %w", err)
	}

	if err := monitorService.UpdateMonitorQuery(ctx, h.cfg.MonitorConfig.ID, newQuery); err != nil {
		return fmt.Errorf("failed to update monitor query: %w", err)
	}

	if err := monitorService.UpdateMonitorCriticalThreshold(ctx, h.cfg.MonitorConfig.ID, h.cfg.MonitorConfig.NewCriticalThreshold); err != nil {
		return fmt.Errorf("failed to update critical threshold: %w", err)
	}

	return nil
}
