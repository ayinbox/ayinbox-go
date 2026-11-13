package ayinbox

import (
	"context"
	"slices"
	"time"
)

// LeadHold pauses one lead in one campaign.
type LeadHold struct {
	Since  time.Time  `json:"since"`
	Until  *time.Time `json:"until,omitempty"`
	Reason string     `json:"reason,omitempty"`
	// Source is out_of_office or manual.
	Source string `json:"source"`
}

// LeadHoldState is a lead's hold in a campaign; Hold is nil when there is none.
type LeadHoldState struct {
	CampaignID string    `json:"campaign_id"`
	ContactID  string    `json:"contact_id"`
	Hold       *LeadHold `json:"hold,omitempty"`
}

// LeadListParams filters the leads of one campaign.
type LeadListParams struct {
	// Cursor is the next_cursor of the previous page.
	Cursor string
	// Limit is the page size. 1 to 100 is recommended; the default is 50.
	Limit int
	// LeadStatus filters by the lead's status in the campaign.
	LeadStatus string
	// Engagement filters by engagement in the campaign.
	Engagement string
}

// LeadPauseParams is the optional body of a lead pause.
type LeadPauseParams struct {
	// Until ends the hold; nil holds until a resume.
	Until  *time.Time `json:"until,omitempty"`
	Reason string     `json:"reason,omitempty"`
}

// CampaignLeadsService manages the contacts attached to a campaign. Leads are
// listed through contact search and added through contact create.
type CampaignLeadsService struct{ c *Client }

func leadSearch(campaignID string, p *LeadListParams) *ContactListParams {
	if p == nil {
		p = &LeadListParams{}
	}
	return &ContactListParams{
		Cursor: p.Cursor,
		Limit:  p.Limit,
		Search: ContactSearch{CampaignIDs: []string{campaignID}, LeadStatus: p.LeadStatus, Engagement: p.Engagement},
	}
}

// List returns one page of a campaign's leads (POST /contacts/search with one
// campaign id). Each row carries CampaignLead. Scope READ_CONTACTS.
func (s *CampaignLeadsService) List(ctx context.Context, campaignID string, params *LeadListParams, opts ...RequestOption) (*ContactList, error) {
	return (&ContactsService{c: s.c}).List(ctx, leadSearch(campaignID, params), opts...)
}

// ListAutoPaging walks every lead of a campaign.
func (s *CampaignLeadsService) ListAutoPaging(campaignID string, params *LeadListParams, opts ...RequestOption) *Pager[Contact] {
	return (&ContactsService{c: s.c}).ListAutoPaging(leadSearch(campaignID, params), opts...)
}

// Add creates or updates contacts and attaches them to the campaign
// (POST /contacts). This can send mail: leads added to an active campaign are
// queued, and a completed campaign starts again on its own. Scope WRITE_CONTACTS.
func (s *CampaignLeadsService) Add(ctx context.Context, campaignID string, contacts []ContactInput, opts ...RequestOption) ([]Contact, error) {
	in := make([]ContactInput, len(contacts))
	for i, ct := range contacts {
		if !slices.Contains(ct.Campaigns, campaignID) {
			ct.Campaigns = append(slices.Clone(ct.Campaigns), campaignID)
		}
		in[i] = ct
	}
	return (&ContactsService{c: s.c}).Create(ctx, in, opts...)
}

// GetHold returns a lead's hold (GET /campaigns/{id}/leads/{contactId}/hold).
// Scope READ_CAMPAIGNS.
func (s *CampaignLeadsService) GetHold(ctx context.Context, campaignID, contactID string, opts ...RequestOption) (*LeadHoldState, error) {
	var out LeadHoldState
	if _, err := s.c.Request(ctx, "GET", pathf("/campaigns/%s/leads/%s/hold", campaignID, contactID), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Pause holds one lead in a campaign (POST /campaigns/{id}/leads/{contactId}/pause).
// params may be nil. Scope WRITE_CAMPAIGNS.
func (s *CampaignLeadsService) Pause(ctx context.Context, campaignID, contactID string, params *LeadPauseParams, opts ...RequestOption) (*LeadHoldState, error) {
	var out LeadHoldState
	if _, err := s.c.Request(ctx, "POST", pathf("/campaigns/%s/leads/%s/pause", campaignID, contactID), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Resume lifts a lead's hold (POST /campaigns/{id}/leads/{contactId}/resume).
// On a completed or paused_undeliverable campaign this can restart it and
// send mail, and needs START_CAMPAIGNS or SEND_CAMPAIGNS. Scope WRITE_CAMPAIGNS.
func (s *CampaignLeadsService) Resume(ctx context.Context, campaignID, contactID string, opts ...RequestOption) (*LeadHoldState, error) {
	var out LeadHoldState
	if _, err := s.c.Request(ctx, "POST", pathf("/campaigns/%s/leads/%s/resume", campaignID, contactID), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
