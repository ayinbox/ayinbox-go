package ayinbox

import (
	"context"
	"encoding/json"
)

// Identity is the member and workspace behind a credential.
type Identity struct {
	UserID           string   `json:"user_id"`
	Email            string   `json:"email"`
	Name             string   `json:"name"`
	FirstName        string   `json:"first_name"`
	LastName         string   `json:"last_name"`
	OrganizationID   string   `json:"organization_id,omitempty"`
	OrganizationName string   `json:"organization_name,omitempty"`
	AuthType         string   `json:"auth_type"`
	Scopes           []string `json:"scopes"`

	// Raw holds the full response body, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the identity and keeps the raw body in Raw.
func (v *Identity) UnmarshalJSON(b []byte) error {
	type alias Identity
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// MeService reads the identity behind the credential.
type MeService struct{ c *Client }

// Get validates the credential and returns who it acts for (GET /me). It needs no scope.
func (s *MeService) Get(ctx context.Context, opts ...RequestOption) (*Identity, error) {
	var out Identity
	if _, err := s.c.Request(ctx, "GET", "/me", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
