package main

import (
    "fmt"
    "time"
    "sort"
    "github.com/nlopes/slack"
)

var slackApi *slack.Client

type SlackClient struct {
    api      *slack.Client
    channel  string
    params   *slack.PostMessageParameters
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
