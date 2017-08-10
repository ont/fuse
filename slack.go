package main

import (
    "fmt"
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
        IconURL: icon_url,
    }

    return &SlackClient{
        api: slackApi,
        channel: channel,
        params: &params,
    }
}

func (s *SlackClient) Good(title string, msg string, details map[string]string) error {
    attachment := slack.Attachment{
        //Title: title,
        //Text: msg,
        Fields: s.makeFields(details),
        Color: "good",
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(
        s.channel,
        fmt.Sprintf("*%s* \n %s", title, msg),
        *s.params,
    )
    return err
}

func (s *SlackClient) Warn(title string, msg string, details map[string]string) error {
    attachment := slack.Attachment{
        Fields: s.makeFields(details),
        Color: "warning",
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(
        s.channel,
        fmt.Sprintf("*%s* \n %s", title, msg),
        *s.params,
    )
    return err
}

func (s *SlackClient) Crit(title string, msg string, details map[string]string) error {
    attachment := slack.Attachment{
        Fields: s.makeFields(details),
        Color: "danger",
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(
        s.channel,
        fmt.Sprintf("*%s* \n %s", title, msg),
        *s.params,
    )
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
