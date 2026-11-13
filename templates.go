package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// Template is a reply template.
type Template struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Name           string    `json:"name"`
	Subject        string    `json:"subject"`
	BodyHTML       string    `json:"body_html"`
	BodyPlain      string    `json:"body_plain"`
	Position       int       `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the template and keeps the raw object in Raw.
func (v *Template) UnmarshalJSON(b []byte) error {
	type alias Template
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// TemplateCreateParams is the body of POST /templates. Name is required.
type TemplateCreateParams struct {
	Name      string `json:"name"`
	Subject   string `json:"subject,omitempty"`
	BodyHTML  string `json:"body_html,omitempty"`
	BodyPlain string `json:"body_plain,omitempty"`
}

// TemplateUpdateParams is the body of PATCH /templates/{id}. A nil field is left unchanged.
type TemplateUpdateParams struct {
	Name      *string `json:"name,omitempty"`
	Subject   *string `json:"subject,omitempty"`
	BodyHTML  *string `json:"body_html,omitempty"`
	BodyPlain *string `json:"body_plain,omitempty"`
}

// TemplatesService manages reply templates.
type TemplatesService struct{ c *Client }

// List returns every template, optionally filtered by q (GET /templates).
// It is not paginated. Scope READ_TEMPLATES.
func (s *TemplatesService) List(ctx context.Context, q string, opts ...RequestOption) ([]Template, error) {
	var out struct {
		Data []Template `json:"data"`
	}
	query := encodeQuery(&struct {
		Q string `url:"q"`
	}{q})
	if _, err := s.c.Request(ctx, "GET", "/templates", query, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Get returns one template (GET /templates/{id}). Scope READ_TEMPLATES.
func (s *TemplatesService) Get(ctx context.Context, id string, opts ...RequestOption) (*Template, error) {
	var out Template
	if _, err := s.c.Request(ctx, "GET", pathf("/templates/%s", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates a template (POST /templates). Scope WRITE_TEMPLATES.
func (s *TemplatesService) Create(ctx context.Context, params *TemplateCreateParams, opts ...RequestOption) (*Template, error) {
	var out Template
	if _, err := s.c.Request(ctx, "POST", "/templates", nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes a template (PATCH /templates/{id}). Scope WRITE_TEMPLATES.
func (s *TemplatesService) Update(ctx context.Context, id string, params *TemplateUpdateParams, opts ...RequestOption) (*Template, error) {
	if params == nil {
		params = &TemplateUpdateParams{}
	}
	var out Template
	if _, err := s.c.Request(ctx, "PATCH", pathf("/templates/%s", id), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a template (DELETE /templates/{id}). Scope WRITE_TEMPLATES.
func (s *TemplatesService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/templates/%s", id), nil, nil, nil, opts...)
	return err
}
