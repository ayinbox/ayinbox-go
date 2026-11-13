package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// Campaign is an outreach campaign.
type Campaign struct {
	ID                        string             `json:"id"`
	UserID                    string             `json:"user_id"`
	OrganizationID            string             `json:"organization_id,omitempty"`
	Name                      string             `json:"name"`
	Description               string             `json:"description"`
	Status                    string             `json:"status"`
	StopOnReply               bool               `json:"stop_on_reply"`
	StopOnMeeting             bool               `json:"stop_on_meeting"`
	OpenTracking              bool               `json:"open_tracking"`
	LinkTracking              bool               `json:"link_tracking"`
	TextOnly                  bool               `json:"text_only"`
	UnsubscribeHeader         bool               `json:"unsubscribe_header"`
	RiskyEmails               bool               `json:"risky_emails"`
	DailyLimit                int                `json:"daily_limit"`
	UnsubscribeMode           string             `json:"unsubscribe_mode"`
	CC                        []string           `json:"cc"`
	BCC                       []string           `json:"bcc"`
	StartDate                 *time.Time         `json:"start_date"`
	EndDate                   *time.Time         `json:"end_date"`
	Timezone                  string             `json:"timezone"`
	EffectiveTimezone         string             `json:"effective_timezone"`
	Days                      int                `json:"days"`
	StartTime                 string             `json:"start_time"`
	EndTime                   string             `json:"end_time"`
	ScheduleWindows           [][]ScheduleWindow `json:"schedule_windows"`
	EmailTags                 []string           `json:"email_tags"`
	Folders                   []string           `json:"folders"`
	ContactOrderBy            string             `json:"contact_order_by"`
	ContactOrderDir           string             `json:"contact_order_dir"`
	ContactOrderField         string             `json:"contact_order_field,omitempty"`
	SenderStrategy            string             `json:"sender_strategy"`
	RotationMode              string             `json:"rotation_mode"`
	Senders                   []CampaignSender   `json:"senders,omitempty"`
	RampEnabled               bool               `json:"ramp_enabled"`
	RampStart                 int                `json:"ramp_start"`
	RampIncrement             int                `json:"ramp_increment"`
	RampCeiling               int                `json:"ramp_ceiling"`
	RampLevel                 int                `json:"ramp_level"`
	RampLevelDate             *time.Time         `json:"ramp_level_date,omitempty"`
	ESPMatchMode              string             `json:"esp_match_mode"`
	MaxNewLeadsPerDay         int                `json:"max_new_leads_per_day"`
	PrioritizeNewLeads        bool               `json:"prioritize_new_leads"`
	Continuous                bool               `json:"continuous"`
	ReviewFirstEmails         bool               `json:"review_first_emails"`
	EntryDelayMinutes         int                `json:"entry_delay_minutes"`
	IdleSince                 *time.Time         `json:"idle_since,omitempty"`
	GuardrailEnabled          bool               `json:"guardrail_enabled"`
	GuardrailBounceRateMax    float64            `json:"guardrail_bounce_rate_max"`
	GuardrailComplaintRateMax float64            `json:"guardrail_complaint_rate_max"`
	GuardrailReplyRateMin     float64            `json:"guardrail_reply_rate_min"`
	GuardrailMinSample        int                `json:"guardrail_min_sample"`
	GuardrailWindowDays       int                `json:"guardrail_window_days"`
	GuardrailTrippedAt        *time.Time         `json:"guardrail_tripped_at,omitempty"`
	GuardrailReason           string             `json:"guardrail_reason,omitempty"`
	TrackingDomain            string             `json:"tracking_domain"`
	TrackingDomainVerified    bool               `json:"tracking_domain_verified"`
	TrackingDomainVerifiedAt  *time.Time         `json:"tracking_domain_verified_at,omitempty"`
	UTMTracking               bool               `json:"utm_tracking"`
	UTMSource                 string             `json:"utm_source"`
	UTMMedium                 string             `json:"utm_medium"`
	UTMCampaign               string             `json:"utm_campaign"`
	LastStatusChangeAt        *time.Time         `json:"last_status_change_at,omitempty"`
	CreatedAt                 time.Time          `json:"created_at"`
	UpdatedAt                 time.Time          `json:"updated_at"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the campaign and keeps the raw object in Raw.
func (v *Campaign) UnmarshalJSON(b []byte) error {
	type alias Campaign
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// ScheduleWindow is a sending window in minutes since local midnight.
type ScheduleWindow struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// CampaignSender is one mailbox a campaign sends from.
type CampaignSender struct {
	EmailAccountID string     `json:"email_account_id"`
	Weight         int        `json:"weight"`
	Enabled        bool       `json:"enabled"`
	LastSentAt     *time.Time `json:"last_sent_at,omitempty"`
}

// CampaignSenderInput names a mailbox to send from when creating a campaign.
type CampaignSenderInput struct {
	EmailAccountID string `json:"email_account_id"`
	Weight         *int   `json:"weight,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

// CampaignListParams filters GET /campaigns.
type CampaignListParams struct {
	// Q searches campaign names.
	Q string `url:"q"`
	// Status is one of draft, active, paused, completed.
	Status string `url:"status"`
	// Folder is a folder uuid.
	Folder string `url:"folder"`
	// Cursor is the next_cursor of the previous page.
	Cursor string `url:"cursor"`
	// Limit is the page size. 1 to 100 is recommended; the default is 50.
	Limit int `url:"limit"`
}

// StepInput is one step created inside POST /campaigns.
type StepInput struct {
	Name        string `json:"name,omitempty"`
	Subject     string `json:"subject,omitempty"`
	BodyPlain   string `json:"body_plain,omitempty"`
	BodyHTML    string `json:"body_html,omitempty"`
	BodySync    *bool  `json:"body_sync,omitempty"`
	BodyCode    *bool  `json:"body_code,omitempty"`
	WaitAfter   *int   `json:"wait_after,omitempty"`
	ThreadReply *bool  `json:"thread_reply,omitempty"`
}

// CampaignCreateParams is the body of POST /campaigns. Only Name is required.
type CampaignCreateParams struct {
	Name               string                `json:"name"`
	Description        *string               `json:"description,omitempty"`
	StopOnReply        *bool                 `json:"stop_on_reply,omitempty"`
	StopOnMeeting      *bool                 `json:"stop_on_meeting,omitempty"`
	OpenTracking       *bool                 `json:"open_tracking,omitempty"`
	LinkTracking       *bool                 `json:"link_tracking,omitempty"`
	TextOnly           *bool                 `json:"text_only,omitempty"`
	UnsubscribeHeader  *bool                 `json:"unsubscribe_header,omitempty"`
	RiskyEmails        *bool                 `json:"risky_emails,omitempty"`
	DailyLimit         *int                  `json:"daily_limit,omitempty"`
	UnsubscribeMode    *string               `json:"unsubscribe_mode,omitempty"`
	CC                 []string              `json:"cc,omitempty"`
	BCC                []string              `json:"bcc,omitempty"`
	StartDate          *time.Time            `json:"start_date,omitempty"`
	EndDate            *time.Time            `json:"end_date,omitempty"`
	Timezone           *string               `json:"timezone,omitempty"`
	Days               *int                  `json:"days,omitempty"`
	StartTime          *string               `json:"start_time,omitempty"`
	EndTime            *string               `json:"end_time,omitempty"`
	ScheduleWindows    [][]ScheduleWindow    `json:"schedule_windows,omitempty"`
	EmailTagIDs        []string              `json:"email_tag_ids,omitempty"`
	FolderIDs          []string              `json:"folder_ids,omitempty"`
	SenderStrategy     *string               `json:"sender_strategy,omitempty"`
	RotationMode       *string               `json:"rotation_mode,omitempty"`
	Senders            []CampaignSenderInput `json:"senders,omitempty"`
	RampEnabled        *bool                 `json:"ramp_enabled,omitempty"`
	RampStart          *int                  `json:"ramp_start,omitempty"`
	RampIncrement      *int                  `json:"ramp_increment,omitempty"`
	RampCeiling        *int                  `json:"ramp_ceiling,omitempty"`
	ESPMatchMode       *string               `json:"esp_match_mode,omitempty"`
	MaxNewLeadsPerDay  *int                  `json:"max_new_leads_per_day,omitempty"`
	PrioritizeNewLeads *bool                 `json:"prioritize_new_leads,omitempty"`
	EntryDelayMinutes  *int                  `json:"entry_delay_minutes,omitempty"`
	Continuous         *bool                 `json:"continuous,omitempty"`
	ReviewFirstEmails  *bool                 `json:"review_first_emails,omitempty"`
	TrackingDomain     *string               `json:"tracking_domain,omitempty"`
	UTMTracking        *bool                 `json:"utm_tracking,omitempty"`
	UTMSource          *string               `json:"utm_source,omitempty"`
	UTMMedium          *string               `json:"utm_medium,omitempty"`
	UTMCampaign        *string               `json:"utm_campaign,omitempty"`
	// Steps are created in order and connected in order.
	Steps []StepInput `json:"steps,omitempty"`
	// Variants are A/B variants of the first step. Their shape is not
	// documented, so they are passed through as given.
	Variants any `json:"variants,omitempty"`
	// AdvancedOverrides is passed through as given.
	AdvancedOverrides map[string]any `json:"advanced_overrides,omitempty"`
}

// CampaignUpdateParams is the body of PATCH /campaigns/{id}. A nil field is
// left unchanged. StartDate and EndDate take ClearTime() to clear them.
//
// Setting Status to "active" runs the full start path and can send mail; it
// needs START_CAMPAIGNS or SEND_CAMPAIGNS in addition to WRITE_CAMPAIGNS.
type CampaignUpdateParams struct {
	Name                      *string            `json:"name,omitempty"`
	Description               *string            `json:"description,omitempty"`
	Status                    *string            `json:"status,omitempty"`
	StopOnReply               *bool              `json:"stop_on_reply,omitempty"`
	StopOnMeeting             *bool              `json:"stop_on_meeting,omitempty"`
	OpenTracking              *bool              `json:"open_tracking,omitempty"`
	LinkTracking              *bool              `json:"link_tracking,omitempty"`
	TextOnly                  *bool              `json:"text_only,omitempty"`
	UnsubscribeHeader         *bool              `json:"unsubscribe_header,omitempty"`
	RiskyEmails               *bool              `json:"risky_emails,omitempty"`
	DailyLimit                *int               `json:"daily_limit,omitempty"`
	UnsubscribeMode           *string            `json:"unsubscribe_mode,omitempty"`
	CC                        []string           `json:"cc,omitempty"`
	BCC                       []string           `json:"bcc,omitempty"`
	StartDate                 *NullTime          `json:"start_date,omitempty"`
	EndDate                   *NullTime          `json:"end_date,omitempty"`
	Timezone                  *string            `json:"timezone,omitempty"`
	Days                      *int               `json:"days,omitempty"`
	StartTime                 *string            `json:"start_time,omitempty"`
	EndTime                   *string            `json:"end_time,omitempty"`
	ScheduleWindows           [][]ScheduleWindow `json:"schedule_windows,omitempty"`
	EmailTags                 []string           `json:"email_tags,omitempty"`
	Folders                   []string           `json:"folders,omitempty"`
	ContactOrderBy            *string            `json:"contact_order_by,omitempty"`
	ContactOrderDir           *string            `json:"contact_order_dir,omitempty"`
	ContactOrderField         *string            `json:"contact_order_field,omitempty"`
	SenderStrategy            *string            `json:"sender_strategy,omitempty"`
	RotationMode              *string            `json:"rotation_mode,omitempty"`
	RampEnabled               *bool              `json:"ramp_enabled,omitempty"`
	RampStart                 *int               `json:"ramp_start,omitempty"`
	RampIncrement             *int               `json:"ramp_increment,omitempty"`
	RampCeiling               *int               `json:"ramp_ceiling,omitempty"`
	ESPMatchMode              *string            `json:"esp_match_mode,omitempty"`
	MaxNewLeadsPerDay         *int               `json:"max_new_leads_per_day,omitempty"`
	PrioritizeNewLeads        *bool              `json:"prioritize_new_leads,omitempty"`
	EntryDelayMinutes         *int               `json:"entry_delay_minutes,omitempty"`
	Continuous                *bool              `json:"continuous,omitempty"`
	ReviewFirstEmails         *bool              `json:"review_first_emails,omitempty"`
	TrackingDomain            *string            `json:"tracking_domain,omitempty"`
	UTMTracking               *bool              `json:"utm_tracking,omitempty"`
	UTMSource                 *string            `json:"utm_source,omitempty"`
	UTMMedium                 *string            `json:"utm_medium,omitempty"`
	UTMCampaign               *string            `json:"utm_campaign,omitempty"`
	GuardrailEnabled          *bool              `json:"guardrail_enabled,omitempty"`
	GuardrailBounceRateMax    *float64           `json:"guardrail_bounce_rate_max,omitempty"`
	GuardrailComplaintRateMax *float64           `json:"guardrail_complaint_rate_max,omitempty"`
	GuardrailReplyRateMin     *float64           `json:"guardrail_reply_rate_min,omitempty"`
	GuardrailMinSample        *int               `json:"guardrail_min_sample,omitempty"`
	GuardrailWindowDays       *int               `json:"guardrail_window_days,omitempty"`
}

// CampaignStartParams is the optional body of POST /campaigns/{id}/start.
type CampaignStartParams struct {
	// AcknowledgeListRisk launches past the bounce-risk gate.
	AcknowledgeListRisk bool `json:"acknowledge_list_risk,omitempty"`
}

// CampaignStartResult is the answer to a start.
type CampaignStartResult struct {
	Status string `json:"status"`
	// WaitingForLeads is true when the campaign is active with nothing to send yet.
	WaitingForLeads bool `json:"waiting_for_leads"`
}

// CampaignPauseResult is the answer to a pause.
type CampaignPauseResult struct {
	Status string `json:"status"`
}

// CampaignDuplicateParams is the optional body of POST /campaigns/{id}/duplicate.
type CampaignDuplicateParams struct {
	Name string `json:"name,omitempty"`
}

// CampaignsOverview counts campaigns by status.
type CampaignsOverview struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Paused    int `json:"paused"`
	Draft     int `json:"draft"`
	Completed int `json:"completed"`
	Folders   []struct {
		FolderID string `json:"folder_id"`
		Total    int    `json:"total"`
	} `json:"folders"`
}

// CampaignsService manages campaigns. Steps and Leads hang off it.
type CampaignsService struct {
	c *Client
	// Steps manages a campaign's steps.
	Steps *CampaignStepsService
	// Leads manages the contacts attached to a campaign.
	Leads *CampaignLeadsService
}

// List returns one page of campaigns (GET /campaigns). Scope READ_CAMPAIGNS.
func (s *CampaignsService) List(ctx context.Context, params *CampaignListParams, opts ...RequestOption) (*Page[Campaign], error) {
	var out Page[Campaign]
	if _, err := s.c.Request(ctx, "GET", "/campaigns", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAutoPaging walks every campaign matching params.
func (s *CampaignsService) ListAutoPaging(params *CampaignListParams, opts ...RequestOption) *Pager[Campaign] {
	q := encodeQuery(params)
	return Paginate[Campaign](s.c, "GET", "/campaigns", q, nil, opts...)
}

// Get returns one campaign (GET /campaigns/{id}). Scope READ_CAMPAIGNS.
func (s *CampaignsService) Get(ctx context.Context, id string, opts ...RequestOption) (*Campaign, error) {
	var out Campaign
	if _, err := s.c.Request(ctx, "GET", pathf("/campaigns/%s", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates a draft campaign (POST /campaigns). It never sends mail.
// Scope WRITE_CAMPAIGNS.
func (s *CampaignsService) Create(ctx context.Context, params *CampaignCreateParams, opts ...RequestOption) (*Campaign, error) {
	var out Campaign
	if _, err := s.c.Request(ctx, "POST", "/campaigns", nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes a campaign (PATCH /campaigns/{id}). Scope WRITE_CAMPAIGNS;
// a Status of active, completed or paused_undeliverable also needs
// START_CAMPAIGNS or SEND_CAMPAIGNS, and active can send mail.
func (s *CampaignsService) Update(ctx context.Context, id string, params *CampaignUpdateParams, opts ...RequestOption) (*Campaign, error) {
	if params == nil {
		params = &CampaignUpdateParams{}
	}
	var out Campaign
	if _, err := s.c.Request(ctx, "PATCH", pathf("/campaigns/%s", id), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete permanently deletes a campaign (DELETE /campaigns/{id}). Scope WRITE_CAMPAIGNS.
func (s *CampaignsService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/campaigns/%s", id), nil, nil, nil, opts...)
	return err
}

// Start launches a campaign (POST /campaigns/{id}/start). It sends real mail.
// Scope START_CAMPAIGNS or SEND_CAMPAIGNS. params may be nil.
func (s *CampaignsService) Start(ctx context.Context, id string, params *CampaignStartParams, opts ...RequestOption) (*CampaignStartResult, error) {
	var body any
	if params != nil {
		body = params
	}
	var out CampaignStartResult
	if _, err := s.c.Request(ctx, "POST", pathf("/campaigns/%s/start", id), nil, body, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Pause stops an active campaign (POST /campaigns/{id}/stop); it becomes
// paused. It sends nothing. Scope WRITE_CAMPAIGNS, START_CAMPAIGNS or SEND_CAMPAIGNS.
func (s *CampaignsService) Pause(ctx context.Context, id string, opts ...RequestOption) (*CampaignPauseResult, error) {
	var out CampaignPauseResult
	if _, err := s.c.Request(ctx, "POST", pathf("/campaigns/%s/stop", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Duplicate copies a campaign into a new draft (POST /campaigns/{id}/duplicate).
// params may be nil. Scope WRITE_CAMPAIGNS.
func (s *CampaignsService) Duplicate(ctx context.Context, id string, params *CampaignDuplicateParams, opts ...RequestOption) (*Campaign, error) {
	var body any
	if params != nil {
		body = params
	}
	var out Campaign
	if _, err := s.c.Request(ctx, "POST", pathf("/campaigns/%s/duplicate", id), nil, body, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Overview counts campaigns by status (GET /campaigns-overview). Scope READ_CAMPAIGNS.
func (s *CampaignsService) Overview(ctx context.Context, opts ...RequestOption) (*CampaignsOverview, error) {
	var out CampaignsOverview
	if _, err := s.c.Request(ctx, "GET", "/campaigns-overview", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
