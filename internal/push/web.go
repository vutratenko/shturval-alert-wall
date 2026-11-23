package push

import (
	"context"
	"fmt"
	"io"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type WebSender struct {
	PublicKey  string
	PrivateKey string
	Subject    string
	Client     webpush.HTTPClient
}

func (s *WebSender) Send(ctx context.Context, sub Subscription, payload []byte) error {
	subject := s.Subject
	if subject == "" {
		subject = "mailto:alerts@sion2k.ru"
	}
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{Auth: sub.Keys.Auth, P256dh: sub.Keys.P256dh},
	}, &webpush.Options{
		Subscriber:      subject,
		VAPIDPublicKey:  s.PublicKey,
		VAPIDPrivateKey: s.PrivateKey,
		TTL:             3600,
		Topic:           "shturval-alerts",
		Urgency:         webpush.UrgencyHigh,
		HTTPClient:      s.Client,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return ErrGone
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("push status %d", resp.StatusCode)
	}
	return nil
}
