package main

import (
	//"io"

	"fmt"
	"sort"
	"time"

	"github.com/nlopes/slack"
	//"github.com/davecgh/go-spew/spew"
)

var slackApi *slack.Client

type SlackClient struct {
	BotUsername string
	IconURL     string

	api     *slack.Client
	channel string
	iconUrl string
	reports map[string]Message
}

func NewSlackClient(channel string, token string, iconUrl string) *SlackClient {
	slackApi := slack.New(token)

	return &SlackClient{
		api:     slackApi,
		channel: channel,
		iconUrl: iconUrl,
		reports: make(map[string]Message),
	}
}

func (s *SlackClient) Good(msg Message) error {
	msg.Level = MSG_LVL_GOOD
	params := s.messageToParams(msg)

	_, _, err := s.api.PostMessage(s.channel, "", *params)
	return err
}

func (s *SlackClient) Warn(msg Message) error {
	msg.Level = MSG_LVL_WARN
	params := s.messageToParams(msg)

	_, _, err := s.api.PostMessage(s.channel, "", *params)
	return err
}

func (s *SlackClient) Crit(msg Message) error {
	msg.Level = MSG_LVL_CRIT
	params := s.messageToParams(msg)

	_, _, err := s.api.PostMessage(s.channel, "", *params)
	return err
}

func (s *SlackClient) messageToParams(msg Message) *slack.PostMessageParameters {
	params := s.makeDefaultSlackParams()

	params.Attachments = s.messageToAttachments(msg)

	return params
}

func (s *SlackClient) makeDefaultSlackParams() *slack.PostMessageParameters {
	return &slack.PostMessageParameters{
		Username: "fuse",
		IconURL:  s.iconUrl,
	}
}

func (s *SlackClient) messageToAttachments(msg Message) []slack.Attachment {
	attachment := slack.Attachment{
		Color:      s.levelToColor(msg.Level),
		Title:      msg.Title,
		Text:       msg.Body,
		MarkdownIn: []string{"text"},
		Fields:     s.makeFields(msg.Details),
		Footer:     fmt.Sprintf("%s | %s", msg.From, time.Now().Format("2006-01-02 15:04:05")),
		FooterIcon: msg.IconUrl,
	}
	return []slack.Attachment{attachment}
}

func (s *SlackClient) makeFields(details map[string]string) []slack.AttachmentField {
	if details == nil {
		return nil
	}

	fields := make([]slack.AttachmentField, 0, len(details))

	keys := make([]string, 0, len(details))
	for key, _ := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, name := range keys {
		fields = append(fields, slack.AttachmentField{
			Title: name,
			Value: details[name],
			Short: true,
		})
	}

	return fields
}

func (s *SlackClient) Report(reportId string, msg Message) error {
	s.reports[reportId] = msg
	return nil
}

func (s *SlackClient) Resolve(reportId string) error {
	delete(s.reports, reportId)
	return nil
}

func (s *SlackClient) levelToColor(level int) string {
	switch level {
	case MSG_LVL_GOOD:
		return "good"
	case MSG_LVL_WARN:
		return "warning"
	case MSG_LVL_CRIT:
		return "danger"
	default:
		return ""
	}
}
