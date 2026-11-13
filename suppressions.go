package ayinbox

import (
	"context"
	"time"
)

// Suppression is an address or domain that no mail goes to.
type Suppression struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	// Email is an address or a domain.
	Email string `json:"email"`
	// Kind is email or domain.
	Kind       string         `json:"kind"`
	Reason     string         `json:"reason"`
	Source     string         `json:"source"`
	CampaignID string         `json:"campaign_id,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// SuppressionListParams filters GET /suppressions.
type SuppressionListParams struct {
	Q string `url:"q"`
	// Limit is 1 to 200; the default is 50.
	Limit  int    `url:"limit"`
	Cursor string `url:"cursor"`
}

// SuppressionEntry is one address or domain to suppress.
type SuppressionEntry struct {
	Value  string `json:"value"`
	Reason string `json:"reason,omitempty"`
}

// SuppressionAddParams is the body of POST /suppressions.
type SuppressionAddParams struct {
	Entries []SuppressionEntry `json:"entries"`
	// Reason applies to entries that carry none.
	Reason string `json:"reason,omitempty"`
}

// SuppressionAddResult reports what an add did.
type SuppressionAddResult struct {
	Added   int      `json:"added"`
	Skipped []string `json:"skipped"`
}

// SuppressionsService manages the suppression list.
type SuppressionsService struct{ c *Client }

// List returns one page of suppressions (GET /suppressions). Scope READ_CONTACTS.
func (s *SuppressionsService) List(ctx context.Context, params *SuppressionListParams, opts ...RequestOption) (*Page[Suppression], error) {
	var out Page[Suppression]
	if _, err := s.c.Request(ctx, "GET", "/suppressions", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAutoPaging walks every suppression matching params.
func (s *SuppressionsService) ListAutoPaging(params *SuppressionListParams, opts ...RequestOption) *Pager[Suppression] {
	return Paginate[Suppression](s.c, "GET", "/suppressions", encodeQuery(params), nil, opts...)
}

// Add suppresses addresses or domains (POST /suppressions). Scope WRITE_CONTACTS.
func (s *SuppressionsService) Add(ctx context.Context, params *SuppressionAddParams, opts ...RequestOption) (*SuppressionAddResult, error) {
	var out SuppressionAddResult
	if _, err := s.c.Request(ctx, "POST", "/suppressions", nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remove lifts a suppression (DELETE /suppressions/{id}), so mail can reach
// that recipient again. Scope WRITE_CONTACTS.
func (s *SuppressionsService) Remove(ctx context.Context, id string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/suppressions/%s", id), nil, nil, nil, opts...)
	return err
}
