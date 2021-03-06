package slack

import (
	//"io"

	"fmt"
	"sort"
	"time"

	"fuse/pkg/domain"

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
	reports map[string]domain.Message
}

func NewSlackClient(channel string, token string, iconUrl string) *SlackClient {
	slackApi := slack.New(token)

	return &SlackClient{
		api:     slackApi,
		channel: channel,
		iconUrl: iconUrl,
		reports: make(map[string]domain.Message),
	}
}

func (s *SlackClient) GetName() string {
	return "slack"
}

func (s *SlackClient) Good(msg domain.Message) error {
	msg.Level = domain.MSG_LVL_GOOD
	options := s.messageToOptions(msg)

	_, _, err := s.api.PostMessage(s.channel, options...)
	return err
}

func (s *SlackClient) Warn(msg domain.Message) error {
	msg.Level = domain.MSG_LVL_WARN
	options := s.messageToOptions(msg)

	_, _, err := s.api.PostMessage(s.channel, options...)
	return err
}

func (s *SlackClient) Crit(msg domain.Message) error {
	msg.Level = domain.MSG_LVL_CRIT
	options := s.messageToOptions(msg)

	_, _, err := s.api.PostMessage(s.channel, options...)
	return err
}

func (s *SlackClient) messageToOptions(msg domain.Message) []slack.MsgOption {
	opts := []slack.MsgOption{
		slack.MsgOptionUsername("fuse"),
		slack.MsgOptionIconURL(s.iconUrl),
		slack.MsgOptionAttachments(s.messageToAttachments(msg)...),
	}

	return opts
}

func (s *SlackClient) messageToAttachments(msg domain.Message) []slack.Attachment {
	return []slack.Attachment{
		slack.Attachment{
			Color:      s.levelToColor(msg.Level),
			Title:      msg.Title,
			Text:       msg.Body,
			MarkdownIn: []string{"text"},
			Fields:     s.makeFields(msg),
			Footer:     fmt.Sprintf("%s | %s", msg.From, time.Now().Format("2006-01-02 15:04:05")),
			FooterIcon: msg.IconUrl,
		},
	}
}

func (s *SlackClient) makeFields(msg domain.Message) []slack.AttachmentField {
	if len(msg.Details) == 0 && len(msg.Args) == 0 {
		return nil
	}

	fields := make([]slack.AttachmentField, 0)

	keys := make([]string, 0)
	kv := make(map[string]interface{})
	for key, value := range msg.Details {
		keys = append(keys, key)
		kv[key] = value
	}

	args := ""
	for key, value := range msg.Args {
		args += fmt.Sprintf("%s = \"%s\" \n", key, value)
	}
	keys = append(keys, "args")
	kv["args"] = args

	sort.Strings(keys)

	for _, name := range keys {
		fields = append(fields, slack.AttachmentField{
			Title: name,
			Value: fmt.Sprintf("%v", kv[name]),
			Short: true,
		})
	}

	return fields
}

func (s *SlackClient) Report(reportId string, msg domain.Message) error {
	s.reports[reportId] = msg
	return nil
}

func (s *SlackClient) Resolve(reportId string) error {
	delete(s.reports, reportId)
	return nil
}

func (s *SlackClient) levelToColor(level int) string {
	switch level {
	case domain.MSG_LVL_GOOD:
		return "good"
	case domain.MSG_LVL_WARN:
		return "warning"
	case domain.MSG_LVL_CRIT:
		return "danger"
	default:
		return ""
	}
}
