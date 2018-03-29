package main

import (
	//"io"
	"encoding/json"
	"fmt"
	"github.com/nlopes/slack"
	"net/http"
	"sort"
	"strings"
	"time"
	//"github.com/davecgh/go-spew/spew"
)

var slackApi *slack.Client

type SlackClient struct {
	api      *slack.Client
	channel  string
	icon_url string
	reports  map[string]Message
}

func NewSlackClient(channel string, token string, icon_url string) *SlackClient {
	slackApi := slack.New(token)

	return &SlackClient{
		api:      slackApi,
		channel:  channel,
		icon_url: icon_url,
		reports:  make(map[string]Message),
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

	attachment := slack.Attachment{
		Color:      s.levelToColor(msg.Level),
		Title:      msg.Title,
		Text:       msg.Body,
		MarkdownIn: []string{"text"},
		Fields:     s.makeFields(msg.Details),
		Footer:     fmt.Sprintf("%s | %s", msg.From, time.Now().Format("2006-01-02 15:04:05")),
		FooterIcon: msg.IconUrl,
	}

	params.Attachments = []slack.Attachment{attachment}

	return params
}

func (s *SlackClient) makeDefaultSlackParams() *slack.PostMessageParameters {
	return &slack.PostMessageParameters{
		Username: "fuse",
		IconURL:  s.icon_url,
	}
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

/*
 * Implements HTTP callback server for slash-command in slack.
 */
func (s *SlackClient) Start() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cmd := r.FormValue("text")

		params := s.ProcessCmd(cmd)

		// output json answer
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(params)
	})

	return http.ListenAndServe(":7777", nil)
}

func (s *SlackClient) ProcessCmd(cmd string) *slack.PostMessageParameters {
	arr := strings.Split(cmd, " ")
	switch arr[0] {
	case "help":
		return s.ProcessHelpCmd()
	case "list":
		return s.ProcessListCmd(arr[1:])
	case "show":
		return s.ProcessShowCmd(arr[1:])
	default:
		return s.ProcessHelpCmd()
	}
}

func (s *SlackClient) ProcessHelpCmd() *slack.PostMessageParameters {
	params := s.makeDefaultSlackParams()

	params.Text = "Usage:\n" +
		"`/fuse help` — this help\n" +
		"`/fuse list` — list all active reports\n" +
		"`/fuse show {report-id}` — show one particular report from list"

	return params
}

func (s *SlackClient) ProcessListCmd(options []string) *slack.PostMessageParameters {
	params := s.makeDefaultSlackParams()

	if len(s.reports) == 0 {
		params.Text = "No issue reports! All works!"
		return params
	}

	attachments := make([]slack.Attachment, 0, len(s.reports))

	for id, report := range s.reports {
		attachments = append(attachments, slack.Attachment{
			Text:       fmt.Sprintf("`%s` — %s\n", id, report.Title),
			Color:      s.levelToColor(report.Level),
			MarkdownIn: []string{"text"},
		})
	}

	params.Attachments = attachments

	return params
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

func (s *SlackClient) ProcessShowCmd(options []string) *slack.PostMessageParameters {
	id := options[0]
	report, ok := s.reports[id]

	if !ok {
		params := s.makeDefaultSlackParams()
		params.Text = fmt.Sprintf("Can't find report with id: `%s`", id)
		return params
	}

	return s.messageToParams(report)
}
