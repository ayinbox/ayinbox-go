package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// DashboardAnalytics is the answer of GET /analytics/dashboard. Nested shapes
// the API does not document are kept untyped.
type DashboardAnalytics struct {
	Period         string           `json:"period"`
	OverallStats   OverallStats     `json:"overall_stats"`
	RecentActivity []map[string]any `json:"recent_activity"`
	TopCampaigns   []any            `json:"top_campaigns"`
	AccountHealth  map[string]any   `json:"account_health"`
	DailyTrend     []any            `json:"daily_trend"`
	CapacityToday  map[string]any   `json:"capacity_today,omitempty"`
	Scope          map[string]any   `json:"scope,omitempty"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the dashboard and keeps the raw object in Raw.
func (v *DashboardAnalytics) UnmarshalJSON(b []byte) error {
	type alias DashboardAnalytics
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// OverallStats are the headline numbers of the dashboard.
type OverallStats struct {
	TotalEmailsSent int64   `json:"total_emails_sent"`
	TotalOpens      int64   `json:"total_opens"`
	MachineOpens    int64   `json:"machine_opens"`
	TotalClicks     int64   `json:"total_clicks"`
	MachineClicks   int64   `json:"machine_clicks"`
	TotalReplies    int64   `json:"total_replies"`
	TotalBounces    int64   `json:"total_bounces"`
	OpenRate        float64 `json:"open_rate"`
	ClickRate       float64 `json:"click_rate"`
	ReplyRate       float64 `json:"reply_rate"`
	BounceRate      float64 `json:"bounce_rate"`
	ActiveCampaigns int64   `json:"active_campaigns"`
	ActiveAccounts  int64   `json:"active_accounts"`
}

// DashboardParams filters GET /analytics/dashboard.
type DashboardParams struct {
	// Period is 7d (default), 30d or 90d.
	Period      string   `url:"period"`
	CampaignIDs []string `url:"campaign_ids"`
	FolderIDs   []string `url:"folder_ids"`
}

// DateRangeParams bounds a campaign analytics read. The format the API
// expects is not documented; YYYY-MM-DD is the likely one.
type DateRangeParams struct {
	From string `url:"from"`
	To   string `url:"to"`
}

// CampaignAnalytics is the answer of GET /analytics/campaigns/{id}.
type CampaignAnalytics struct {
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	DateRange  struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"date_range"`
	Summary CampaignSummary `json:"summary"`
	// Steps, DailyStats and Engagement are kept untyped.
	Steps      []any          `json:"steps"`
	DailyStats []any          `json:"daily_stats,omitempty"`
	Engagement map[string]any `json:"engagement,omitempty"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the analytics and keeps the raw object in Raw.
func (v *CampaignAnalytics) UnmarshalJSON(b []byte) error {
	type alias CampaignAnalytics
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// CampaignSummary is a campaign's totals.
type CampaignSummary struct {
	TotalContacts   int64 `json:"total_contacts"`
	EmailsSent      int64 `json:"emails_sent"`
	EmailsPending   int64 `json:"emails_pending"`
	UniqueOpens     int64 `json:"unique_opens"`
	MachineOpens    int64 `json:"machine_opens"`
	UniqueClicks    int64 `json:"unique_clicks"`
	MachineClicks   int64 `json:"machine_clicks"`
	Replies         int64 `json:"replies"`
	PositiveReplies int64 `json:"positive_replies"`
	InterestedLeads int64 `json:"interested_leads"`
	Bounces         int64 `json:"bounces"`
	Unsubscribes    int64 `json:"unsubscribes"`
}

// AccountAnalyticsParams filters GET /analytics/accounts.
type AccountAnalyticsParams struct {
	// EmailIDs limits the list to up to 200 mailboxes.
	EmailIDs []string `url:"email_ids"`
	// Limit is 1 to 1000; the default is 1000.
	Limit  int    `url:"limit"`
	Cursor string `url:"cursor"`
}

// AnalyticsService reads analytics. Every route needs READ_ANALYTICS.
type AnalyticsService struct{ c *Client }

// Dashboard returns the workspace dashboard (GET /analytics/dashboard).
func (s *AnalyticsService) Dashboard(ctx context.Context, params *DashboardParams, opts ...RequestOption) (*DashboardAnalytics, error) {
	var out DashboardAnalytics
	if _, err := s.c.Request(ctx, "GET", "/analytics/dashboard", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// Campaign returns one campaign's analytics (GET /analytics/campaigns/{id}).
func (s *AnalyticsService) Campaign(ctx context.Context, id string, params *DateRangeParams, opts ...RequestOption) (*CampaignAnalytics, error) {
	var out CampaignAnalytics
	if _, err := s.c.Request(ctx, "GET", pathf("/analytics/campaigns/%s", id), encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// CampaignDaily returns one campaign's daily rows
// (GET /analytics/campaigns/{id}/daily). The row shape is not documented,
// so rows are returned as raw JSON.
func (s *AnalyticsService) CampaignDaily(ctx context.Context, id string, params *DateRangeParams, opts ...RequestOption) ([]json.RawMessage, error) {
	var out struct {
		Data []json.RawMessage `json:"data"`
	}
	if _, err := s.c.Request(ctx, "GET", pathf("/analytics/campaigns/%s/daily", id), encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Accounts returns one page of per-mailbox health and usage
// (GET /analytics/accounts). The row shape is not documented, so rows are raw JSON.
func (s *AnalyticsService) Accounts(ctx context.Context, params *AccountAnalyticsParams, opts ...RequestOption) (*Page[json.RawMessage], error) {
	var out Page[json.RawMessage]
	if _, err := s.c.Request(ctx, "GET", "/analytics/accounts", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// AccountsAutoPaging walks every row of Accounts.
func (s *AnalyticsService) AccountsAutoPaging(params *AccountAnalyticsParams, opts ...RequestOption) *Pager[json.RawMessage] {
	return Paginate[json.RawMessage](s.c, "GET", "/analytics/accounts", encodeQuery(params), nil, opts...)
}

// Account returns one mailbox's status detail (GET /analytics/accounts/{id}).
// The shape is not documented, so it is returned as a map.
func (s *AnalyticsService) Account(ctx context.Context, id string, opts ...RequestOption) (map[string]any, error) {
	var out map[string]any
	if _, err := s.c.Request(ctx, "GET", pathf("/analytics/accounts/%s", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}
