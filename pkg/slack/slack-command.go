package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlopes/slack"
)

/*
 * Implements HTTP callback server for slash-command in slack.
 */
func (s *SlackClient) ConfigureHTTP() {
	// TODO: configure http via DI; replace http with iris?
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cmd := r.FormValue("text")

		params := s.ProcessCmd(cmd)

		// output json answer
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(params)
	})
}

func (s *SlackClient) ProcessCmd(cmd string) *slack.Msg {
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

func (s *SlackClient) ProcessHelpCmd() *slack.Msg {
	params := s.makeDefaultSlackMsg()

	params.Text = "Usage:\n" +
		"`/fuse help` — this help\n" +
		"`/fuse list` — list all active reports\n" +
		"`/fuse show {report-id}` — show one particular report from list"

	return params
}

func (s *SlackClient) ProcessListCmd(options []string) *slack.Msg {
	params := s.makeDefaultSlackMsg()

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

func (s *SlackClient) ProcessShowCmd(options []string) *slack.Msg {
	id := options[0]
	report, ok := s.reports[id]

	params := s.makeDefaultSlackMsg()

	if !ok {
		params.Text = fmt.Sprintf("Can't find report with id: `%s`", id)
		return params
	}

	params.Attachments = s.messageToAttachments(report)
	return params
}

func (s *SlackClient) makeDefaultSlackMsg() *slack.Msg {
	return &slack.Msg{
		Username: "fuse",
		Icons: &slack.Icon{
			IconURL: s.iconUrl,
		},
	}
}
