package main

import "fmt"
import "github.com/nlopes/slack"

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

func (s *SlackClient) Good(name string, msg string) error {
    attachment := slack.Attachment{
        Title: fmt.Sprintf("Service \"%s\"", name),
        Text: msg,
        Fields: []slack.AttachmentField{
            slack.AttachmentField{
                Title: "Service",
                Value: name,
                Short: true,
            },
            slack.AttachmentField{
                Title: "State",
                Value: "GOOD",
                Short: true,
            },
        },
        Color: "good",
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(s.channel, "", *s.params)
    return err
}

func (s *SlackClient) Warn(name string, msg string) error {
    attachment := slack.Attachment{
        Title: fmt.Sprintf("Service \"%s\"", name),
        Text: msg,
        Fields: []slack.AttachmentField{
            slack.AttachmentField{
                Title: "Service",
                Value: name,
                Short: true,
            },
            slack.AttachmentField{
                Title: "State",
                Value: "WARN",
                Short: true,
            },
        },
        Color: "warning",
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(s.channel, "", *s.params)
    return err
}

func (s *SlackClient) Crit(name string, msg string) error {
    attachment := slack.Attachment{
        Title: fmt.Sprintf("Service \"%s\"", name),
        Text: msg,
        Fields: []slack.AttachmentField{
            slack.AttachmentField{
                Title: "Service",
                Value: name,
                Short: true,
            },
            slack.AttachmentField{
                Title: "State",
                Value: "CRITICAL",
                Short: true,
            },
        },
        Color: "danger",
    }
    s.params.Attachments = []slack.Attachment{attachment}

    _, _, err := s.api.PostMessage(s.channel, "", *s.params)
    return err
}
