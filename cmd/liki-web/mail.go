package main

import (
	"context"

	"github.com/ml8s/liki-web/internal/email"
	"github.com/ml8s/liki-web/internal/payment"
)

type paymentEmailSender struct {
	client *email.Client
}

func (s paymentEmailSender) Send(ctx context.Context, msg payment.EmailMessage) error {
	return s.client.SendReport(ctx, msg.To, msg.Subject, msg.Body)
}
