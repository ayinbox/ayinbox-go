package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// Contact is a person in the workspace.
type Contact struct {
	ID                      string            `json:"id"`
	FirstName               string            `json:"first_name"`
	LastName                string            `json:"last_name"`
	Email                   string            `json:"email"`
	Company                 string            `json:"company"`
	Phone                   string            `json:"phone"`
	CustomFields            map[string]string `json:"custom_fields"`
	Subscribed              bool              `json:"subscribed"`
	Campaigns               []ContactCampaign `json:"campaigns"`
	Categories              []Label           `json:"categories"`
	VerificationStatus      string            `json:"verification_status"`
	VerificationReason      string            `json:"verification_reason"`
	IsCatchAll              bool              `json:"is_catch_all"`
	VerificationCheckedAt   *time.Time        `json:"verification_checked_at,omitempty"`
	VerificationSource      string            `json:"verification_source"`
	VerificationProvider    string            `json:"verification_provider"`
	VerificationSubStatus   string            `json:"verification_sub_status"`
	VerificationConfidence  int               `json:"verification_confidence"`
	VerificationRequestedAt *time.Time        `json:"verification_requested_at,omitempty"`
	MailHost                string            `json:"mail_host"`
	ESPProvider             string            `json:"esp_provider"`
	ESPResolvedAt           *time.Time        `json:"esp_resolved_at,omitempty"`
	ConsentGivenAt          *time.Time        `json:"consent_given_at,omitempty"`
	ConsentSource           string            `json:"consent_source,omitempty"`
	// LeadScore holds grade, score and more; its exact fields are not documented.
	LeadScore map[string]any `json:"lead_score,omitempty"`
	// CampaignLead is set on rows of a search with exactly one campaign id.
	CampaignLead *ContactCampaignProgress `json:"campaign_lead,omitempty"`
	// Outreach is set on search rows; its fields are not documented.
	Outreach  map[string]any `json:"outreach,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the contact and keeps the raw object in Raw.
func (v *Contact) UnmarshalJSON(b []byte) error {
	type alias Contact
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// ContactDetail is a contact as GET /contacts/{id} returns it.
type ContactDetail struct {
	Contact
	// Engagement, Suppression and Verification are not documented field by
	// field, so they are kept untyped.
	Engagement     map[string]any `json:"engagement,omitempty"`
	Suppression    map[string]any `json:"suppression,omitempty"`
	Verification   map[string]any `json:"verification,omitempty"`
	Source         string         `json:"source"`
	SourceDetail   string         `json:"source_detail"`
	FirstSeenAt    *time.Time     `json:"first_seen_at,omitempty"`
	ConsentCountry string         `json:"consent_country,omitempty"`
	// ConsentRequired is true when the contact's country needs recorded consent.
	ConsentRequired bool `json:"consent_required"`
}

// UnmarshalJSON decodes the contact and its detail fields.
func (v *ContactDetail) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.Contact); err != nil {
		return err
	}
	var extra struct {
		Engagement      map[string]any `json:"engagement"`
		Suppression     map[string]any `json:"suppression"`
		Verification    map[string]any `json:"verification"`
		Source          string         `json:"source"`
		SourceDetail    string         `json:"source_detail"`
		FirstSeenAt     *time.Time     `json:"first_seen_at"`
		ConsentCountry  string         `json:"consent_country"`
		ConsentRequired bool           `json:"consent_required"`
	}
	if err := json.Unmarshal(b, &extra); err != nil {
		return err
	}
	v.Engagement, v.Suppression, v.Verification = extra.Engagement, extra.Suppression, extra.Verification
	v.Source, v.SourceDetail, v.FirstSeenAt = extra.Source, extra.SourceDetail, extra.FirstSeenAt
	v.ConsentCountry, v.ConsentRequired = extra.ConsentCountry, extra.ConsentRequired
	return nil
}

// ContactCampaign is a campaign a contact belongs to.
type ContactCampaign struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Label is a contact category or unibox label.
type Label struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
}

// ContactCampaignProgress is a contact's progress in one campaign.
type ContactCampaignProgress struct {
	// Status is pending, active, completed, replied, bounced, failed,
	// unsubscribed or another value; treat unknown values as opaque.
	Status         string           `json:"status"`
	Sent           int              `json:"sent"`
	Opened         int              `json:"opened"`
	MachineOpened  int              `json:"machine_opened"`
	Clicked        int              `json:"clicked"`
	Replied        int              `json:"replied"`
	Bounced        int              `json:"bounced"`
	LastActivityAt *time.Time       `json:"last_activity_at,omitempty"`
	CurrentStep    string           `json:"current_step,omitempty"`
	FailureReason  string           `json:"failure_reason,omitempty"`
	Sender         string           `json:"sender,omitempty"`
	Hold           *LeadHold        `json:"hold,omitempty"`
	CC             []map[string]any `json:"cc,omitempty"`
}

// CustomFieldFilter filters contacts on one custom field.
type CustomFieldFilter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	// Type is equal, starts_with, ends_with or contains.
	Type string `json:"type"`
}

// ContactSearch is the body of POST /contacts/search. Every field is optional.
type ContactSearch struct {
	Query              string              `json:"query,omitempty"`
	CustomFieldFilters []CustomFieldFilter `json:"custom_field_filters,omitempty"`
	// CampaignIDs matches contacts in ALL of these campaigns.
	CampaignIDs []string `json:"campaign_ids,omitempty"`
	// LeadStatus and Engagement need exactly one campaign id.
	LeadStatus         string     `json:"lead_status,omitempty"`
	Engagement         string     `json:"engagement,omitempty"`
	CategoryIDs        []string   `json:"category_ids,omitempty"`
	SegmentIDs         []string   `json:"segment_ids,omitempty"`
	MinCampaigns       *int       `json:"min_campaigns,omitempty"`
	MaxCampaigns       *int       `json:"max_campaigns,omitempty"`
	Subscribed         *bool      `json:"subscribed,omitempty"`
	VerificationStatus string     `json:"verification_status,omitempty"`
	MailHosts          []string   `json:"mail_hosts,omitempty"`
	LeadGrades         []string   `json:"lead_grades,omitempty"`
	CreatedAfter       *time.Time `json:"created_after,omitempty"`
	CreatedBefore      *time.Time `json:"created_before,omitempty"`
	UpdatedAfter       *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore      *time.Time `json:"updated_before,omitempty"`
	SortBy             string     `json:"sort_by,omitempty"`
	Reverse            bool       `json:"reverse,omitempty"`
}

// ContactListParams is the query and body of POST /contacts/search.
type ContactListParams struct {
	// Cursor is the next_cursor of the previous page.
	Cursor string `url:"cursor"`
	// Limit is the page size. 1 to 100 is recommended; the default is 50.
	Limit int `url:"limit"`
	// Category is one label uuid.
	Category string `url:"category"`
	// Search is sent as the request body, unchanged on every page.
	Search ContactSearch `url:"-"`
}

// ContactList is one page of a contact search.
type ContactList struct {
	Data       []Contact  `json:"data"`
	Pagination Pagination `json:"pagination"`
	// Counts appears on the first page only.
	Counts map[string]any `json:"counts,omitempty"`
	// LeadCounts holds per-status totals on the first page of a search with
	// one campaign id.
	LeadCounts map[string]any `json:"lead_counts,omitempty"`
}

// ContactInput is one contact to create or upsert. Email is required.
type ContactInput struct {
	Email                string            `json:"email"`
	FirstName            string            `json:"first_name,omitempty"`
	LastName             string            `json:"last_name,omitempty"`
	Company              string            `json:"company,omitempty"`
	Phone                string            `json:"phone,omitempty"`
	Campaigns            []string          `json:"campaigns,omitempty"`
	Categories           []string          `json:"categories,omitempty"`
	Segments             []string          `json:"segments,omitempty"`
	CustomFields         map[string]string `json:"custom_fields,omitempty"`
	Subscribed           *bool             `json:"subscribed,omitempty"`
	VerificationStatus   string            `json:"verification_status,omitempty"`
	VerificationProvider string            `json:"verification_provider,omitempty"`
}

// ContactUpdateParams is the body of PATCH /contacts/{id}. A nil field is left unchanged.
type ContactUpdateParams struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Company   *string `json:"company,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	// CustomFields replaces the contact's custom fields.
	CustomFields map[string]string `json:"custom_fields,omitempty"`
	Subscribed   *bool             `json:"subscribed,omitempty"`
	// Campaigns replaces the contact's campaign list when not nil.
	Campaigns []string `json:"campaigns,omitempty"`
	// Categories replaces the contact's labels when not nil.
	Categories []string `json:"categories,omitempty"`
	// AddCategories and RemoveCategories are ignored when Categories is sent.
	AddCategories    []string `json:"add_categories,omitempty"`
	RemoveCategories []string `json:"remove_categories,omitempty"`
}

// ContactBulkDelete selects the contacts DELETE /contacts removes: either
// IDs, or All with optional Filters and Exclude.
type ContactBulkDelete struct {
	IDs     []string
	All     bool
	Filters *ContactSearch
	Exclude []string
}

// MarshalJSON encodes IDs as a bare array, or the selection object when All is set.
func (b ContactBulkDelete) MarshalJSON() ([]byte, error) {
	if !b.All {
		ids := b.IDs
		if ids == nil {
			ids = []string{}
		}
		return json.Marshal(ids)
	}
	return json.Marshal(struct {
		All     bool           `json:"all"`
		Filters *ContactSearch `json:"filters,omitempty"`
		Exclude []string       `json:"exclude,omitempty"`
	}{true, b.Filters, b.Exclude})
}

// ContactsService manages contacts.
type ContactsService struct{ c *Client }

// List searches contacts and returns one page (POST /contacts/search). It
// only reads and is retried like a GET. params may be nil. Scope READ_CONTACTS.
func (s *ContactsService) List(ctx context.Context, params *ContactListParams, opts ...RequestOption) (*ContactList, error) {
	if params == nil {
		params = &ContactListParams{}
	}
	var out ContactList
	o := append(append([]RequestOption(nil), opts...), withRetrySafe())
	if _, err := s.c.Request(ctx, "POST", "/contacts/search", encodeQuery(params), params.Search, &out, o...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAutoPaging walks every contact matching params, resending the same
// search body on every page.
func (s *ContactsService) ListAutoPaging(params *ContactListParams, opts ...RequestOption) *Pager[Contact] {
	if params == nil {
		params = &ContactListParams{}
	}
	return Paginate[Contact](s.c, "POST", "/contacts/search", encodeQuery(params), params.Search, opts...)
}

// Get returns one contact with its detail fields (GET /contacts/{id}). Scope READ_CONTACTS.
func (s *ContactsService) Get(ctx context.Context, id string, opts ...RequestOption) (*ContactDetail, error) {
	var out ContactDetail
	if _, err := s.c.Request(ctx, "GET", pathf("/contacts/%s", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates or updates contacts by email address (POST /contacts), up
// to 10,240 per call. Adding contacts to an active campaign can send mail.
// Scope WRITE_CONTACTS.
func (s *ContactsService) Create(ctx context.Context, contacts []ContactInput, opts ...RequestOption) ([]Contact, error) {
	if contacts == nil {
		contacts = []ContactInput{}
	}
	var out []Contact
	if _, err := s.c.Request(ctx, "POST", "/contacts", nil, contacts, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// Update changes a contact (PATCH /contacts/{id}). Scope WRITE_CONTACTS.
func (s *ContactsService) Update(ctx context.Context, id string, params *ContactUpdateParams, opts ...RequestOption) (*Contact, error) {
	if params == nil {
		params = &ContactUpdateParams{}
	}
	var out Contact
	if _, err := s.c.Request(ctx, "PATCH", pathf("/contacts/%s", id), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a contact (DELETE /contacts/{id}). Scope WRITE_CONTACTS.
func (s *ContactsService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/contacts/%s", id), nil, nil, nil, opts...)
	return err
}

// BulkDelete deletes many contacts (DELETE /contacts with a JSON body).
// Scope BULK_CONTACTS.
func (s *ContactsService) BulkDelete(ctx context.Context, sel ContactBulkDelete, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", "/contacts", nil, sel, nil, opts...)
	return err
}
