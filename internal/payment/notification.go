package payment

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/money"
)

type EmailMessage struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

type NotificationKind string

const (
	NotificationCustomerReceipt NotificationKind = "customer_receipt"
	NotificationAdminNewOrder   NotificationKind = "admin_new_order"
)

type Notification struct {
	ID              string
	PaymentIntentID string
	Kind            NotificationKind
	To              string
	Subject         string
	Body            string
	Status          string
	RetryCount      int
	LastError       string
	CreatedAt       time.Time
	SentAt          *time.Time
}

var ErrNotificationNotFound = errors.New("payment: notification not found")

type NotificationStore struct {
	db *sql.DB
}

type paidOrderNotice struct {
	Locale          string
	ProductName     string
	OrderID         string
	PaymentIntentID string
	CustomerEmail   string
	Amount          string
	Provider        string
	PaidAt          string
	ContinueURL     string
}

var customerReceiptTemplate = template.Must(template.New("customer_receipt").Parse(
	`<p>付款已完成，正在为你开通 {{.ProductName}}。</p>
<p>订单号：{{.OrderID}}<br>
金额：{{.Amount}}<br>
支付时间：{{.PaidAt}}</p>
<p><a href="{{.ContinueURL}}">继续使用服务</a></p>`))

var customerReceiptTemplateHant = template.Must(template.New("customer_receipt_hant").Parse(
	`<p>付款已完成，正在為你開通 {{.ProductName}}。</p>
<p>訂單號：{{.OrderID}}<br>
金額：{{.Amount}}<br>
支付時間：{{.PaidAt}}</p>
<p><a href="{{.ContinueURL}}">繼續使用服務</a></p>`))

var customerReceiptTemplateEn = template.Must(template.New("customer_receipt_en").Parse(
	`<p>Payment received. We are preparing access to {{.ProductName}}.</p>
<p>Order: {{.OrderID}}<br>
Amount: {{.Amount}}<br>
Paid at: {{.PaidAt}}</p>
<p><a href="{{.ContinueURL}}">Continue</a></p>`))

var customerReceiptSubjects = map[string]string{
	"zh-Hans": "订单确认 — ",
	"zh-Hant": "訂單確認 — ",
	"en":      "Order confirmation — ",
}

var adminNewOrderTemplate = template.Must(template.New("admin_new_order").Parse(
	`<p>新订单已支付。</p>
<p>产品：{{.ProductName}}<br>
订单号：{{.OrderID}}<br>
Payment Intent：{{.PaymentIntentID}}<br>
用户邮箱：{{.CustomerEmail}}<br>
金额：{{.Amount}}<br>
支付方式：{{.Provider}}<br>
支付时间：{{.PaidAt}}</p>`))

func newPaidOrderNotice(intent PaymentIntent, flow ProductFlow, baseURL string) paidOrderNotice {
	paidAt := time.Now().UTC()
	if intent.PaidAt != nil {
		paidAt = intent.PaidAt.UTC()
	}
	return paidOrderNotice{
		ProductName:     flow.DisplayName(notificationLocale(intent.Locale)),
		OrderID:         intent.BusinessOrderID,
		PaymentIntentID: intent.ID,
		CustomerEmail:   intent.Email,
		Amount:          money.FormatDisplay(intent.AmountMinor, intent.Currency),
		Provider:        providerDisplayName(intent.Provider),
		PaidAt:          paidAt.Format("2006-01-02 15:04:05 MST"),
		ContinueURL:     absoluteNotificationURL(baseURL, flow.SuccessRedirect(intent)),
		Locale:          notificationLocale(intent.Locale),
	}
}

// absoluteNotificationURL turns a product flow's HTTP redirect into an email-
// safe absolute URL. SuccessRedirect intentionally remains relative for HTTP.
func absoluteNotificationURL(baseURL, location string) string {
	if location == "" {
		return strings.TrimSpace(baseURL)
	}
	reference, err := url.Parse(location)
	if err != nil {
		return location
	}
	if reference.IsAbs() {
		return reference.String()
	}
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.String() == "" {
		return location
	}
	return base.ResolveReference(reference).String()
}

func providerDisplayName(provider Provider) string {
	switch provider {
	case ProviderDodo:
		return "Dodo"
	case ProviderXunhu:
		return "XunHuPay"
	default:
		return string(provider)
	}
}

func notificationLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en", "en-us", "en-gb":
		return "en"
	case "zh-hans", "zh-cn", "zh-sg":
		return "zh-Hans"
	default:
		return "zh-Hant"
	}
}

func customerReceiptTemplateFor(locale string) *template.Template {
	switch locale {
	case "en":
		return customerReceiptTemplateEn
	case "zh-Hans":
		return customerReceiptTemplate
	default:
		return customerReceiptTemplateHant
	}
}

func buildCustomerReceipt(notice paidOrderNotice) (EmailMessage, error) {
	var body bytes.Buffer
	if err := customerReceiptTemplateFor(notice.Locale).Execute(&body, notice); err != nil {
		return EmailMessage{}, err
	}
	subjectPrefix := customerReceiptSubjects[notice.Locale]
	if subjectPrefix == "" {
		subjectPrefix = customerReceiptSubjects["zh-Hant"]
	}
	return EmailMessage{
		To:      notice.CustomerEmail,
		Subject: subjectPrefix + notice.ProductName + " | liki.hk",
		Body:    body.String(),
	}, nil
}

func buildAdminNewOrder(notice paidOrderNotice, to string) (EmailMessage, error) {
	var body bytes.Buffer
	if err := adminNewOrderTemplate.Execute(&body, notice); err != nil {
		return EmailMessage{}, err
	}
	return EmailMessage{
		To:      to,
		Subject: "新订单 — " + notice.ProductName,
		Body:    body.String(),
	}, nil
}

func NewNotificationStore(db *sql.DB) *NotificationStore { return &NotificationStore{db: db} }

func (s *NotificationStore) Enqueue(ctx context.Context, intentID string, kind NotificationKind, msg EmailMessage) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO payment_notifications (id, payment_intent_id, kind, to_email, subject, body, status, created_at)
VALUES (?,?,?,?,?,?,'pending',?)`,
		uuid.NewString(), intentID, string(kind), msg.To, msg.Subject, msg.Body,
		time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("payment: enqueue notification: %w", err)
	}
	return nil
}

func (s *NotificationStore) EnqueueOnce(ctx context.Context, intentID string, kind NotificationKind, msg EmailMessage) error {
	// Single statement: the unique index (payment_intent_id, kind) is the
	// concurrency-safe authority, so a racing webhook either inserts or
	// idempotently no-ops — no SELECT/INSERT window.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO payment_notifications (id, payment_intent_id, kind, to_email, subject, body, status, created_at)
VALUES (?,?,?,?,?,?,'pending',?)
ON CONFLICT(payment_intent_id, kind) DO NOTHING`,
		uuid.NewString(), intentID, string(kind), msg.To, msg.Subject, msg.Body, now); err != nil {
		return fmt.Errorf("payment: enqueue notification: %w", err)
	}
	return nil
}

func (s *NotificationStore) Migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS payment_notifications (
    id TEXT PRIMARY KEY,
    payment_intent_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    to_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TEXT NOT NULL,
    sent_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_payment_notifications_status
    ON payment_notifications(status, created_at);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return err
	}
	// The application check prevents ordinary duplicates; the index is the
	// concurrency-safe authority. Keep the newest row if legacy data conflicts.
	if _, err := s.db.ExecContext(ctx, `
DELETE FROM payment_notifications
WHERE rowid NOT IN (
    SELECT MAX(rowid) FROM payment_notifications
    GROUP BY payment_intent_id, kind
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_notifications_intent_kind
    ON payment_notifications(payment_intent_id, kind);
`); err != nil {
		return err
	}
	return nil
}

func (s *NotificationStore) Pending(ctx context.Context, limit int) ([]Notification, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, payment_intent_id, kind, to_email, subject, body, status, retry_count, last_error, created_at, sent_at
FROM payment_notifications
WHERE status='pending'
ORDER BY created_at
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Notification
	for rows.Next() {
		var n Notification
		var kind, status, createdAt string
		var lastError, sentAt sql.NullString
		if err := rows.Scan(&n.ID, &n.PaymentIntentID, &kind, &n.To, &n.Subject, &n.Body,
			&status, &n.RetryCount, &lastError, &createdAt, &sentAt); err != nil {
			return nil, err
		}
		n.Kind = NotificationKind(kind)
		n.Status = status
		n.LastError = lastError.String
		if parsed, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
			n.CreatedAt = parsed
		}
		if sentAt.Valid {
			if parsed, parseErr := time.Parse(time.RFC3339, sentAt.String); parseErr == nil {
				n.SentAt = &parsed
			}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *NotificationStore) DeleteSentBefore(ctx context.Context, older time.Duration) (int64, error) {
	before := time.Now().UTC().Add(-older).Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `DELETE FROM payment_notifications WHERE status='sent' AND sent_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *NotificationStore) MarkSent(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE payment_notifications SET status='sent', sent_at=? WHERE id=?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *NotificationStore) MarkFailed(ctx context.Context, id, message string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE payment_notifications
SET status=CASE WHEN retry_count + 1 >= 5 THEN 'failed' ELSE 'pending' END,
    retry_count=retry_count+1,
    last_error=?
WHERE id=?`, message, id)
	return err
}
