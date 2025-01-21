package service

import "github.com/slack-go/slack"

type ISlackClient interface {
    OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
    PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

type SlackService struct {
    client ISlackClient
}

func NewSlackService(client ISlackClient) *SlackService {
    return &SlackService{client: client}
}

func (c *SlackService) OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
    return c.client.OpenView(triggerID, view)
}

func (c *SlackService) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
    return c.client.PostMessage(channelID, options...)
}
