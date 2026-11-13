package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// WebhookEndpoint is a URL that receives webhook deliveries.
type WebhookEndpoint struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	URL            string `json:"url"`
	Description    string `json:"description"`
	// EventTypes is empty to receive every non-firehose event.
	EventTypes          []string   `json:"event_types"`
	Enabled             bool       `json:"enabled"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt       *time.Time `json:"last_failure_at,omitempty"`
	LastFailureReason   string     `json:"last_failure_reason,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	OAuthApplicationID  string     `json:"oauth_application_id,omitempty"`
	CreatedBy           string     `json:"created_by,omitempty"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	OwnershipConfirmed  bool       `json:"ownership_confirmed"`
	AutoDisabledAt      *time.Time `json:"auto_disabled_at,omitempty"`
	DisabledReason      string     `json:"disabled_reason,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	// Secret is set only on the answers of Create (and RotateSecret returns it alone).
	Secret string `json:"secret,omitempty"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the endpoint and keeps the raw object in Raw.
func (v *WebhookEndpoint) UnmarshalJSON(b []byte) error {
	type alias WebhookEndpoint
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// EventType describes one webhook event type.
type EventType struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Firehose    bool   `json:"firehose"`
	Deprecated  bool   `json:"deprecated"`
}

// WebhookList is the answer of GET /webhooks.
type WebhookList struct {
	Endpoints  []WebhookEndpoint `json:"endpoints"`
	EventTypes []EventType       `json:"event_types"`
}

// WebhookParams is the body of a webhook create or update. URL is required.
// An update replaces the whole endpoint: send the full desired state.
type WebhookParams struct {
	// URL must be HTTPS and publicly routable.
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	// Enabled defaults to true when nil.
	Enabled *bool `json:"enabled,omitempty"`
}

// Delivery is one webhook delivery attempt record.
type Delivery struct {
	ID             string `json:"id"`
	EndpointID     string `json:"endpoint_id"`
	OrganizationID string `json:"organization_id"`
	EventType      string `json:"event_type"`
	EventID        string `json:"event_id"`
	// Payload is the delivered event envelope.
	Payload             json.RawMessage `json:"payload"`
	Status              string          `json:"status"`
	AttemptCount        int             `json:"attempt_count"`
	MaxAttempts         int             `json:"max_attempts"`
	NextAttemptAt       *time.Time      `json:"next_attempt_at"`
	LastAttemptAt       *time.Time      `json:"last_attempt_at,omitempty"`
	ResponseStatus      *int            `json:"response_status,omitempty"`
	ResponseBodyExcerpt string          `json:"response_body_excerpt,omitempty"`
	ErrorReason         string          `json:"error_reason,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// DeliveryListParams filters GET /webhooks/deliveries.
type DeliveryListParams struct {
	// Status is pending, in_flight, delivered, failed or abandoned.
	Status    string `url:"status"`
	EventType string `url:"event_type"`
	// EndpointID limits the list to one endpoint.
	EndpointID string `url:"endpoint_id"`
	// Limit is 1 to 200; the default is 50.
	Limit  int    `url:"limit"`
	Cursor string `url:"cursor"`
}

// WebhooksService manages webhook endpoints. Every route needs WEBHOOKS.
type WebhooksService struct{ c *Client }

// List returns every endpoint and the event types (GET /webhooks).
func (s *WebhooksService) List(ctx context.Context, opts ...RequestOption) (*WebhookList, error) {
	var out WebhookList
	if _, err := s.c.Request(ctx, "GET", "/webhooks", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create adds an endpoint (POST /webhooks). The answer holds the signing
// secret, shown only once; a retry with the same idempotency key answers
// 409 idempotent_secret_withheld.
func (s *WebhooksService) Create(ctx context.Context, params *WebhookParams, opts ...RequestOption) (*WebhookEndpoint, error) {
	var out WebhookEndpoint
	if _, err := s.c.Request(ctx, "POST", "/webhooks", nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update replaces an endpoint (PATCH /webhooks/{id}). Omitted fields are not
// kept: a nil Enabled becomes true and EventTypes is overwritten.
func (s *WebhooksService) Update(ctx context.Context, id string, params *WebhookParams, opts ...RequestOption) (*WebhookEndpoint, error) {
	var out WebhookEndpoint
	if _, err := s.c.Request(ctx, "PATCH", pathf("/webhooks/%s", id), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes an endpoint (DELETE /webhooks/{id}).
func (s *WebhooksService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/webhooks/%s", id), nil, nil, nil, opts...)
	return err
}

// RotateSecret issues a new signing secret (POST /webhooks/{id}/rotate-secret)
// and returns it. It is shown only once.
func (s *WebhooksService) RotateSecret(ctx context.Context, id string, opts ...RequestOption) (string, error) {
	var out struct {
		Secret string `json:"secret"`
	}
	if _, err := s.c.Request(ctx, "POST", pathf("/webhooks/%s/rotate-secret", id), nil, nil, &out, opts...); err != nil {
		return "", err
	}
	return out.Secret, nil
}

// Verify re-sends the verification challenge (POST /webhooks/{id}/verify)
// and returns the status, "challenge_sent".
func (s *WebhooksService) Verify(ctx context.Context, id string, opts ...RequestOption) (string, error) {
	var out struct {
		Status string `json:"status"`
	}
	if _, err := s.c.Request(ctx, "POST", pathf("/webhooks/%s/verify", id), nil, nil, &out, opts...); err != nil {
		return "", err
	}
	return out.Status, nil
}

// EventTypes lists the webhook event types (GET /webhooks/event-types).
func (s *WebhooksService) EventTypes(ctx context.Context, opts ...RequestOption) ([]EventType, error) {
	var out struct {
		EventTypes []EventType `json:"event_types"`
	}
	if _, err := s.c.Request(ctx, "GET", "/webhooks/event-types", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.EventTypes, nil
}

// ListDeliveries returns one page of deliveries (GET /webhooks/deliveries).
func (s *WebhooksService) ListDeliveries(ctx context.Context, params *DeliveryListParams, opts ...RequestOption) (*Page[Delivery], error) {
	var out Page[Delivery]
	if _, err := s.c.Request(ctx, "GET", "/webhooks/deliveries", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDeliveriesAutoPaging walks every delivery matching params.
func (s *WebhooksService) ListDeliveriesAutoPaging(params *DeliveryListParams, opts ...RequestOption) *Pager[Delivery] {
	return Paginate[Delivery](s.c, "GET", "/webhooks/deliveries", encodeQuery(params), nil, opts...)
}

// VerifySignature checks a delivery's X-Ayinbox-Signature header locally,
// with no HTTP call. It is VerifyWebhookSignature.
func (s *WebhooksService) VerifySignature(payload []byte, header, secret string) error {
	return VerifyWebhookSignature(payload, header, secret)
}
