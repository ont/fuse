package main

import (
    "io"
    "fmt"
    "time"
    "sort"
    "strings"
    "net/http"
    "github.com/nlopes/slack"
    //"github.com/davecgh/go-spew/spew"
)

var slackApi *slack.Client

type SlackClient struct {
    api      *slack.Client
    channel  string
    params   *slack.PostMessageParameters
    reports  map[string]Message
}


func NewSlackClient(channel string, token string, icon_url string) *SlackClient {
    slackApi := slack.New(token)

    params := slack.PostMessageParameters{
        Username: "fuse",
        IconURL: icon_url,
    }

    return &SlackClient{
        api: slackApi,
        channel: channel,
        params: &params,
    }
}

func (s *SlackClient) Good(msg Message) error {
    attachment := slack.Attachment{
        Title: msg.Title,
        Text: msg.Body,
        MarkdownIn: []string{"text"},
        Fields: s.makeFields(msg.Details),
        Color: "good",
        Footer: fmt.Sprintf("%s | %s", msg.From, time.Now().Format("2006-01-02 15:04:05")),
        FooterIcon: msg.IconUrl,
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(s.channel, "", *s.params)
    return err
}

func (s *SlackClient) Warn(msg Message) error {
    // TODO: refactor code duplication
    attachment := slack.Attachment{
        Title: msg.Title,
        Text: msg.Body,
        MarkdownIn: []string{"text"},
        Fields: s.makeFields(msg.Details),
        Color: "warning",
        Footer: fmt.Sprintf("%s | %s", msg.From, time.Now().Format("2006-01-02 15:04:05")),
        FooterIcon: msg.IconUrl,
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(s.channel, "", *s.params)
    return err
}

func (s *SlackClient) Crit(msg Message) error {
    // TODO: refactor code duplication
    attachment := slack.Attachment{
        Title: msg.Title,
        Text: msg.Body,
        MarkdownIn: []string{"text"},
        Fields: s.makeFields(msg.Details),
        Color: "danger",
        Footer: fmt.Sprintf("%s | %s", msg.From, time.Now().Format("2006-01-02 15:04:05")),
        FooterIcon: msg.IconUrl,
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(s.channel, "", *s.params)
    return err
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
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
        cmd := r.FormValue("text")

        ans := s.ProcessCmd(cmd)

        io.WriteString(w, ans)
    })

    return http.ListenAndServe(":7777", nil)
}

func (s *SlackClient) ProcessCmd(cmd string) string {
    arr := strings.Split(cmd, " ")
    switch arr[0] {
    case "help": return s.ProcessHelpCmd()
    case "list": return s.ProcessListCmd(arr[1:])
    case "show": return s.ProcessShowCmd(arr[1:])
    default: return s.ProcessHelpCmd()
    }
}

func (s *SlackClient) ProcessHelpCmd() string {
    return "Usage:\n" +
        "`/fuse help` — this help\n" +
        "`/fuse list` — list all active reports\n" +
        "`/fuse show {report-id}` — show one particular report from list"
}

func (s *SlackClient) ProcessListCmd(options []string) string {
    res := ""
    for id, report := range s.reports {
        res += fmt.Sprintf("`%s` — %s\n", id, report.Title)
    }

    if res == "" {
        return "No issue reports! All works!"
    }
    return res
}

func (s *SlackClient) ProcessShowCmd(options []string) string {
    return "some answer"
}
