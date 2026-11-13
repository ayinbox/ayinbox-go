package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// Mailbox is a connected sending mailbox. The API returns more fields than
// this type names; read them from Raw.
type Mailbox struct {
	ID                       string     `json:"id"`
	UserID                   string     `json:"user_id"`
	OrganizationID           string     `json:"organization_id,omitempty"`
	WorkerID                 *string    `json:"worker_id"`
	Email                    string     `json:"email"`
	Name                     string     `json:"name"`
	SignaturePlain           string     `json:"signature_plain"`
	SignatureHTML            string     `json:"signature_html"`
	SendAsEmail              string     `json:"send_as_email"`
	Provider                 string     `json:"provider"`
	Status                   string     `json:"status"`
	MailHost                 string     `json:"mail_host"`
	AuthMethod               string     `json:"auth_method"`
	AvatarURL                string     `json:"avatar_url"`
	LastSyncedAt             *time.Time `json:"last_synced_at"`
	CampaignLimit            int        `json:"campaign_limit"`
	MinWaitTime              int        `json:"min_wait_time"`
	ReplyTo                  string     `json:"reply_to"`
	TrackingDomain           string     `json:"tracking_domain"`
	TrackingDomainVerified   bool       `json:"tracking_domain_verified"`
	AuthState                string     `json:"auth_state"`
	AuthSPF                  bool       `json:"auth_spf"`
	AuthDKIM                 bool       `json:"auth_dkim"`
	AuthDMARC                bool       `json:"auth_dmarc"`
	AuthCheckedAt            *time.Time `json:"auth_checked_at,omitempty"`
	Warmup                   *time.Time `json:"warmup"`
	WarmupPausedAt           *time.Time `json:"warmup_paused_at"`
	WarmupBase               int        `json:"warmup_base"`
	WarmupMax                int        `json:"warmup_max"`
	WarmupIncrease           int        `json:"warmup_increase"`
	WarmupReplyRate          int        `json:"warmup_reply_rate"`
	WarmupPoolType           string     `json:"warmup_pool_type"`
	Timezone                 string     `json:"timezone"`
	SaveToSent               bool       `json:"save_to_sent"`
	RelayFolderMoves         bool       `json:"relay_folder_moves"`
	Tags                     []string   `json:"tags"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
	TrackingDomainVerifiedAt *time.Time `json:"tracking_domain_verified_at,omitempty"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the mailbox and keeps the raw object in Raw.
func (v *Mailbox) UnmarshalJSON(b []byte) error {
	type alias Mailbox
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// MailboxListParams filters GET /emails.
type MailboxListParams struct {
	Q string `url:"q"`
	// Tag is a mailbox tag uuid.
	Tag    string `url:"tag"`
	Cursor string `url:"cursor"`
	// Limit is the page size. 1 to 100 is recommended; the default is 50.
	Limit int `url:"limit"`
}

// SyncState is a mailbox's sync progress. More fields may appear in Raw.
type SyncState struct {
	BackfillStatus string `json:"backfill_status"`
	// BackfillCursor is opaque.
	BackfillCursor      json.RawMessage `json:"backfill_cursor,omitempty"`
	BackfillSynced      int64           `json:"backfill_synced"`
	BackfillSince       *time.Time      `json:"backfill_since,omitempty"`
	BackfillStartedAt   *time.Time      `json:"backfill_started_at,omitempty"`
	BackfillCompletedAt *time.Time      `json:"backfill_completed_at,omitempty"`
	ThrottledUntil      *time.Time      `json:"throttled_until,omitempty"`
	ThrottleReason      string          `json:"throttle_reason,omitempty"`
	Deferred            int64           `json:"deferred"`
	LastSyncedAt        *time.Time      `json:"last_synced_at,omitempty"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the state and keeps the raw object in Raw.
func (v *SyncState) UnmarshalJSON(b []byte) error {
	type alias SyncState
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// SyncPolicy is the sync budget that applies to a mailbox.
type SyncPolicy struct {
	BackfillDays     int      `json:"backfill_days"`
	BackfillMessages int      `json:"backfill_messages"`
	DailyMessages    int      `json:"daily_messages"`
	OrgDailyMessages int      `json:"org_daily_messages"`
	SkipFolders      []string `json:"skip_folders,omitempty"`
}

// MailboxSync is the answer of GET /emails/{id}/sync.
type MailboxSync struct {
	// State is nil before the first sync.
	State       *SyncState `json:"state"`
	Policy      SyncPolicy `json:"policy"`
	SkipFolders []string   `json:"skip_folders"`
	Folders     []struct {
		Name   string `json:"name"`
		Folder string `json:"folder"`
	} `json:"folders"`
}

// MailboxesService reads connected mailboxes (the /emails routes).
type MailboxesService struct{ c *Client }

// List returns one page of mailboxes (GET /emails). Scope READ_EMAILS.
func (s *MailboxesService) List(ctx context.Context, params *MailboxListParams, opts ...RequestOption) (*Page[Mailbox], error) {
	var out Page[Mailbox]
	if _, err := s.c.Request(ctx, "GET", "/emails", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAutoPaging walks every mailbox matching params.
func (s *MailboxesService) ListAutoPaging(params *MailboxListParams, opts ...RequestOption) *Pager[Mailbox] {
	return Paginate[Mailbox](s.c, "GET", "/emails", encodeQuery(params), nil, opts...)
}

// Get returns one mailbox (GET /emails/{id}). Scope READ_EMAILS.
func (s *MailboxesService) Get(ctx context.Context, id string, opts ...RequestOption) (*Mailbox, error) {
	var out Mailbox
	if _, err := s.c.Request(ctx, "GET", pathf("/emails/%s", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSync returns a mailbox's sync state and policy (GET /emails/{id}/sync).
// Scope READ_EMAILS.
func (s *MailboxesService) GetSync(ctx context.Context, id string, opts ...RequestOption) (*MailboxSync, error) {
	var out MailboxSync
	if _, err := s.c.Request(ctx, "GET", pathf("/emails/%s/sync", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}
