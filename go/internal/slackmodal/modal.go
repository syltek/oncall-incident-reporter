package slackmodal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
	"github.com/syltek/oncall-incident-reporter/pkg/errors"
)

// Modal represents a Slack modal with a flexible set of blocks and configurations.
type Modal struct {
	TriggerID string
	View      slack.ModalViewRequest
	Payload   ModalPayload // Holds parsed payload for submitted modals
}

// ModalPayload is the payload received when the modal is submitted.
type ModalPayload struct {
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	View struct {
		ID    string `json:"id"`
		State struct {
			Values map[string]map[string]struct {
				Type           string `json:"type"`
				Value          string `json:"value"`
				SelectedOption struct {
					Value string `json:"value"`
				} `json:"selected_option"`
			} `json:"values"`
		} `json:"state"`
	} `json:"view"`
}

// NewModal initializes a new modal with a title.
func NewModal(title, triggerID string) *Modal {
	return &Modal{
		TriggerID: triggerID,
		View: slack.ModalViewRequest{
			Type:       slack.VTModal,
			CallbackID: "default_modal",
			Title: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: title,
			},
			Blocks: slack.Blocks{},
			Submit: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Submit",
			},
			Close: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Cancel",
			},
		},
	}
}

// AddTextInput adds a simple text input field to the modal.
func (m *Modal) AddTextInput(blockID, label, placeholder string, multiline bool) *Modal {
	inputBlock := slack.NewInputBlock(
		blockID,
		slack.NewTextBlockObject(slack.PlainTextType, label, false, false),
		slack.NewTextBlockObject(slack.PlainTextType, placeholder, false, false),
		slack.NewPlainTextInputBlockElement(slack.NewTextBlockObject(slack.PlainTextType, placeholder, false, false), blockID).WithMultiline(multiline),
	)

	m.View.Blocks.BlockSet = append(m.View.Blocks.BlockSet, inputBlock)
	return m
}

// AddSelectInput adds a static select dropdown input to the modal.
func (m *Modal) AddSelectInput(blockID, label, placeholder string, options []string) *Modal {
	slackOptions := make([]*slack.OptionBlockObject, len(options))
	for i, opt := range options {
		slackOptions[i] = slack.NewOptionBlockObject(
			opt,
			slack.NewTextBlockObject(slack.PlainTextType, opt, false, false),
			nil,
		)
	}

	selectElement := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		slack.NewTextBlockObject(slack.PlainTextType, placeholder, false, false),
		blockID,
		slackOptions...,
	)

	inputBlock := slack.NewInputBlock(
		blockID,
		slack.NewTextBlockObject(slack.PlainTextType, label, false, false),
		nil,
		selectElement,
	)
	m.View.Blocks.BlockSet = append(m.View.Blocks.BlockSet, inputBlock)
	return m
}

// AddDateInput adds a date picker input to the modal.
func (m *Modal) AddDateInput(blockID, label string) *Modal {
	dateElement := slack.NewDatePickerBlockElement(blockID)
	inputBlock := slack.NewInputBlock(
		blockID,
		slack.NewTextBlockObject(slack.PlainTextType, label, false, false),
		nil,
		dateElement,
	)
	m.View.Blocks.BlockSet = append(m.View.Blocks.BlockSet, inputBlock)
	return m
}

// SendModal sends the modal using the Slack API.
func (m *Modal) SendModal(api *slack.Client) error {
	_, err := api.OpenView(m.TriggerID, m.View)
	if err != nil {
		return fmt.Errorf("failed to send modal with trigger ID %s: %w", m.TriggerID, err)
	}

	return nil
}

// ParsePayload extracts the payload from a string
func (m *Modal) ParsePayload(payloadStr string) error {
	payload := ModalPayload{}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return errors.New(http.StatusBadRequest, "Invalid payload format", errors.CategoryClient, err)
	}

	m.Payload = payload
	return nil
}

// ParseField retrieves a specific field value from the parsed payload.
func (m *Modal) ParseField(blockID string) (string, error) {
	if values, ok := m.Payload.View.State.Values[blockID]; ok {
		for _, field := range values {
			if field.Type == "static_select" {
				return field.SelectedOption.Value, nil
			}
			return field.Value, nil
		}
	}
	return "", errors.New(http.StatusBadRequest, fmt.Sprintf("Field %s not found", blockID), errors.CategoryClient, nil)
}

func (m *Modal) GetUsername() string {
	return m.Payload.User.Username
}

// ParseAllFields retrieves all the field values from the parsed modal payload.
func (m *Modal) ParseAllFields() (map[string]string, error) {
	// Create a map to store field values
	fields := make(map[string]string)

	// Iterate over the values in the modal payload and extract the data
	for blockID, valueMap := range m.Payload.View.State.Values {
		for _, field := range valueMap {
			// Check if the field is a select input (dropdown)
			if field.Type == "static_select" {
				fields[blockID] = field.SelectedOption.Value
			} else {
				// Otherwise, take the regular text input value
				fields[blockID] = field.Value
			}
		}
	}

	// If no fields are found, return an error
	if len(fields) == 0 {
		return nil, errors.New(http.StatusBadRequest, "No fields found in modal", errors.CategoryClient, nil)
	}

	// Return the parsed field values
	return fields, nil
}
