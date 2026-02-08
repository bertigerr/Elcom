package connectors

import "elcom/internal"

type MailConnector interface {
	FetchInbox(label string, max int) ([]internal.FetchedMailMessage, error)
}
