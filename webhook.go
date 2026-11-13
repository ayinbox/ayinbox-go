package ayinbox

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// DefaultWebhookTolerance is the largest accepted gap between a webhook's
// signed timestamp and the current time.
const DefaultWebhookTolerance = 300 * time.Second

// SignatureHeader is the header that carries a webhook delivery's signature.
const SignatureHeader = "X-Ayinbox-Signature"

// Errors returned by webhook verification. All of them match
// ErrInvalidSignature with errors.Is.
var (
	// ErrInvalidSignature matches every webhook verification failure.
	ErrInvalidSignature = errors.New("ayinbox: invalid webhook signature")
	// ErrSignatureHeader is returned for a missing or malformed header.
	ErrSignatureHeader = &signatureError{"malformed " + SignatureHeader + " header"}
	// ErrSignatureExpired is returned when the timestamp is outside the tolerance.
	ErrSignatureExpired = &signatureError{"webhook timestamp outside the tolerance"}
	// ErrSignatureMismatch is returned when no v1 signature matches.
	ErrSignatureMismatch = &signatureError{"no matching v1 signature"}
)

type signatureError struct{ msg string }

func (e *signatureError) Error() string { return "ayinbox: invalid webhook signature: " + e.msg }

func (e *signatureError) Is(target error) bool { return target == ErrInvalidSignature }

// WebhookEvent is the envelope of every webhook delivery.
type WebhookEvent struct {
	// ID equals the X-Ayinbox-Event-Id header and is stable across retries.
	ID string `json:"id"`
	// EventType is the event name, such as "campaign.reply_received".
	EventType string `json:"event_type"`
	// OrganizationID is the workspace the event belongs to.
	OrganizationID string `json:"organization_id"`
	// CreatedAt is when the event happened.
	CreatedAt time.Time `json:"created_at"`
	// Data holds the event payload.
	Data json.RawMessage `json:"data"`
}

// VerifyWebhookSignature checks the X-Ayinbox-Signature header of a delivery
// against the raw request body and the endpoint secret (including its whsec_
// prefix), with the default 300 second tolerance. It returns nil when valid.
// Pass the body exactly as received, before any JSON parsing.
func VerifyWebhookSignature(payload []byte, header, secret string) error {
	return VerifyWebhookSignatureAt(payload, header, secret, DefaultWebhookTolerance, time.Now())
}

// VerifyWebhookSignatureAt is VerifyWebhookSignature with an explicit
// tolerance and current time. A tolerance of 0 or less uses the default.
func VerifyWebhookSignatureAt(payload []byte, header, secret string, tolerance time.Duration, now time.Time) error {
	if tolerance <= 0 {
		tolerance = DefaultWebhookTolerance
	}
	var ts string
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	if ts == "" || len(sigs) == 0 {
		return ErrSignatureHeader
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ErrSignatureHeader
	}
	diff := now.Unix() - t
	if diff < 0 {
		diff = -diff
	}
	if time.Duration(diff)*time.Second > tolerance {
		return ErrSignatureExpired
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := []byte(hex.EncodeToString(mac.Sum(nil)))
	for _, s := range sigs {
		if hmac.Equal(expected, []byte(strings.ToLower(s))) {
			return nil
		}
	}
	return ErrSignatureMismatch
}

// ConstructWebhookEvent verifies a delivery with VerifyWebhookSignature and
// then decodes its envelope.
func ConstructWebhookEvent(payload []byte, header, secret string) (*WebhookEvent, error) {
	if err := VerifyWebhookSignature(payload, header, secret); err != nil {
		return nil, err
	}
	var ev WebhookEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
