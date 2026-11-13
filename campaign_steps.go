package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// Step is one step of a campaign sequence.
type Step struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Subject     string  `json:"subject"`
	BodyPlain   string  `json:"body_plain"`
	BodyHTML    string  `json:"body_html"`
	BodySync    bool    `json:"body_sync"`
	BodyCode    bool    `json:"body_code"`
	WaitAfter   int     `json:"wait_after"`
	Position    int     `json:"position"`
	ThreadReply bool    `json:"thread_reply"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	// Conditions is the routing object; {} ends the flow at this step.
	// Its schema is not documented, so it is kept untyped.
	Conditions map[string]any `json:"conditions,omitempty"`
	// Kind is email, action or wait.
	Kind string `json:"kind"`
	// Action is the action node config, kept untyped.
	Action    map[string]any `json:"action,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the step and keeps the raw object in Raw.
func (v *Step) UnmarshalJSON(b []byte) error {
	type alias Step
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// StepParams is the body of a step create or update. Every field is optional.
type StepParams struct {
	Name        *string `json:"name,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	BodyPlain   *string `json:"body_plain,omitempty"`
	BodyHTML    *string `json:"body_html,omitempty"`
	BodySync    *bool   `json:"body_sync,omitempty"`
	BodyCode    *bool   `json:"body_code,omitempty"`
	WaitAfter   *int    `json:"wait_after,omitempty"`
	ThreadReply *bool   `json:"thread_reply,omitempty"`
	// Conditions is passed through as given (schema not documented). nil
	// omits it; map[string]any{} sends {}, which ends the flow at this step.
	Conditions any     `json:"conditions,omitempty"`
	Kind       *string `json:"kind,omitempty"`
	// Action is passed through as given (schema not documented).
	Action any `json:"action,omitempty"`
}

// CampaignStepsService manages a campaign's steps.
type CampaignStepsService struct{ c *Client }

// List returns a campaign's steps in canvas order (GET /campaigns/{id}/steps).
// Scope READ_CAMPAIGNS.
func (s *CampaignStepsService) List(ctx context.Context, campaignID string, opts ...RequestOption) ([]Step, error) {
	var out []Step
	if _, err := s.c.Request(ctx, "GET", pathf("/campaigns/%s/steps", campaignID), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// Create adds a step (POST /campaigns/{id}/steps). params may be nil for a
// blank step. The step is not connected to the previous one; to build a
// linear sequence, pass Steps to Campaigns.Create. Scope WRITE_CAMPAIGNS.
func (s *CampaignStepsService) Create(ctx context.Context, campaignID string, params *StepParams, opts ...RequestOption) (*Step, error) {
	if params == nil {
		params = &StepParams{}
	}
	var out Step
	if _, err := s.c.Request(ctx, "POST", pathf("/campaigns/%s/steps", campaignID), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes a step (PATCH /campaigns/{id}/steps/{stepId}). Scope WRITE_CAMPAIGNS.
func (s *CampaignStepsService) Update(ctx context.Context, campaignID, stepID string, params *StepParams, opts ...RequestOption) (*Step, error) {
	if params == nil {
		params = &StepParams{}
	}
	var out Step
	if _, err := s.c.Request(ctx, "PATCH", pathf("/campaigns/%s/steps/%s", campaignID, stepID), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a step (DELETE /campaigns/{id}/steps/{stepId}). Scope WRITE_CAMPAIGNS.
func (s *CampaignStepsService) Delete(ctx context.Context, campaignID, stepID string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/campaigns/%s/steps/%s", campaignID, stepID), nil, nil, nil, opts...)
	return err
}
