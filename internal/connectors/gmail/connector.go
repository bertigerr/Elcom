package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"elcom/internal"
	"elcom/internal/config"
)

type Connector struct {
	service *gmail.Service
}

func NewConnector(cfg config.Config) (*Connector, error) {
	if err := cfg.Require("GMAIL_CLIENT_ID", cfg.GmailClientID); err != nil {
		return nil, err
	}
	if err := cfg.Require("GMAIL_CLIENT_SECRET", cfg.GmailClientSecret); err != nil {
		return nil, err
	}
	if err := cfg.Require("GMAIL_REFRESH_TOKEN", cfg.GmailRefreshToken); err != nil {
		return nil, err
	}

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GmailClientID,
		ClientSecret: cfg.GmailClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  cfg.GmailRedirectURI,
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	tokenSource := oauthCfg.TokenSource(context.Background(), &oauth2.Token{RefreshToken: cfg.GmailRefreshToken})
	svc, err := gmail.NewService(context.Background(), option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	return &Connector{service: svc}, nil
}

func (c *Connector) FetchInbox(label string, max int) ([]internal.FetchedMailMessage, error) {
	listCall := c.service.Users.Messages.List("me").LabelIds(label).MaxResults(int64(max))
	listResp, err := listCall.Do()
	if err != nil {
		return nil, err
	}

	messages := listResp.Messages
	out := make([]internal.FetchedMailMessage, 0, len(messages))

	for _, msgRef := range messages {
		if msgRef.Id == "" {
			continue
		}

		rawResp, err := c.service.Users.Messages.Get("me", msgRef.Id).Format("raw").Do()
		if err != nil {
			return nil, err
		}
		metaResp, err := c.service.Users.Messages.Get("me", msgRef.Id).Format("metadata").MetadataHeaders("Subject", "From", "Date", "Message-ID").Do()
		if err != nil {
			return nil, err
		}

		if rawResp.Raw == "" {
			continue
		}

		rawBytes, err := decodeBase64URL(rawResp.Raw)
		if err != nil {
			return nil, err
		}

		headers := map[string]string{}
		if metaResp.Payload != nil {
			for _, h := range metaResp.Payload.Headers {
				headers[strings.ToLower(h.Name)] = h.Value
			}
		}

		received := time.Now().UTC().Format(time.RFC3339)
		if dateHeader := headers["date"]; dateHeader != "" {
			if t, err := time.Parse(time.RFC1123Z, dateHeader); err == nil {
				received = t.UTC().Format(time.RFC3339)
			} else if t, err := mailDateFallback(dateHeader); err == nil {
				received = t.UTC().Format(time.RFC3339)
			}
		}

		messageID := headers["message-id"]
		if messageID == "" {
			messageID = msgRef.Id
		}

		out = append(out, internal.FetchedMailMessage{
			Provider:   "gmail",
			MessageID:  messageID,
			Subject:    headers["subject"],
			From:       headers["from"],
			ReceivedAt: received,
			Raw:        rawBytes,
		})
	}

	return out, nil
}

func decodeBase64URL(input string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(input)
	if err == nil {
		return decoded, nil
	}
	decoded, err = base64.URLEncoding.DecodeString(input)
	if err == nil {
		return decoded, nil
	}
	return nil, fmt.Errorf("decode gmail raw payload: %w", err)
}

func mailDateFallback(value string) (time.Time, error) {
	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC850, time.ANSIC}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format")
}
