package imap

import (
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"

	"elcom/internal"
	"elcom/internal/config"
)

type Connector struct {
	host     string
	port     int
	secure   bool
	user     string
	password string
	markSeen bool
}

func NewConnector(cfg config.Config) (*Connector, error) {
	if err := cfg.Require("IMAP_HOST", cfg.IMAPHost); err != nil {
		return nil, err
	}
	if err := cfg.Require("IMAP_USER", cfg.IMAPUser); err != nil {
		return nil, err
	}
	if err := cfg.Require("IMAP_PASSWORD", cfg.IMAPPassword); err != nil {
		return nil, err
	}

	return &Connector{
		host:     cfg.IMAPHost,
		port:     cfg.IMAPPort,
		secure:   cfg.IMAPSecure,
		user:     cfg.IMAPUser,
		password: cfg.IMAPPassword,
		markSeen: cfg.IMAPMarkSeen,
	}, nil
}

func (c *Connector) FetchInbox(label string, max int) ([]internal.FetchedMailMessage, error) {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	var client *imapclient.Client
	var err error
	if c.secure {
		client, err = imapclient.DialTLS(addr, &tls.Config{ServerName: c.host})
	} else {
		client, err = imapclient.Dial(addr)
	}
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	if err := client.Login(c.user, c.password); err != nil {
		return nil, err
	}

	_, err = client.Select(label, false)
	if err != nil {
		return nil, err
	}

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := client.Search(criteria)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	if len(ids) > max {
		ids = ids[len(ids)-max:]
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchInternalDate, imap.FetchUid, section.FetchItem()}
	messages := make(chan *imap.Message, len(ids))
	fetchDone := make(chan error, 1)
	go func() { fetchDone <- client.Fetch(seqset, items, messages) }()

	out := make([]internal.FetchedMailMessage, 0, len(ids))
	for msg := range messages {
		if msg == nil {
			continue
		}
		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		raw, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}

		messageID := ""
		subject := ""
		from := ""
		if msg.Envelope != nil {
			messageID = msg.Envelope.MessageId
			subject = msg.Envelope.Subject
			from = formatAddresses(msg.Envelope.From)
		}
		if messageID == "" {
			messageID = fmt.Sprintf("imap-%d", msg.Uid)
		}

		received := time.Now().UTC().Format(time.RFC3339)
		if !msg.InternalDate.IsZero() {
			received = msg.InternalDate.UTC().Format(time.RFC3339)
		}

		out = append(out, internal.FetchedMailMessage{
			Provider:   "imap",
			MessageID:  messageID,
			Subject:    subject,
			From:       from,
			ReceivedAt: received,
			Raw:        raw,
		})

		if c.markSeen {
			single := new(imap.SeqSet)
			single.AddNum(msg.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.SeenFlag}
			if err := client.Store(single, item, flags, nil); err != nil {
				return nil, err
			}
		}
	}

	if err := <-fetchDone; err != nil {
		return nil, err
	}

	return out, nil
}

func formatAddresses(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a == nil {
			continue
		}
		email := strings.Trim(strings.Join([]string{a.MailboxName, a.HostName}, "@"), "@")
		if a.PersonalName != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", a.PersonalName, email))
		} else {
			parts = append(parts, email)
		}
	}
	return strings.Join(parts, ", ")
}
