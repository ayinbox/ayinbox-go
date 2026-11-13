package ayinbox

import (
	"context"
	"encoding/json"
	"time"
)

// MessagePreview is one row of a unibox list or thread.
type MessagePreview struct {
	ID       string   `json:"id"`
	EmailID  string   `json:"email_id"`
	ThreadID string   `json:"thread_id"`
	FromAddr []string `json:"from_addr"`
	ToAddr   []string `json:"to_addr"`
	Subject  string   `json:"subject"`
	Snippet  string   `json:"snippet"`
	// InternalDate is when the message arrived.
	InternalDate time.Time `json:"internal_date"`
	Seen         bool      `json:"seen"`
	// MessageCount is the number of messages in the thread (list reads).
	MessageCount     int     `json:"message_count"`
	HasUnread        bool    `json:"has_unread"`
	Labels           []Label `json:"labels"`
	AnswersMailboxID string  `json:"answers_mailbox_id,omitempty"`
	CampaignID       string  `json:"campaign_id,omitempty"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the row and keeps the raw object in Raw.
func (v *MessagePreview) UnmarshalJSON(b []byte) error {
	type alias MessagePreview
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// Message is one full message.
type Message struct {
	ID        string    `json:"id"`
	GmailID   string    `json:"gmail_id"`
	UID       int64     `json:"uid"`
	EmailID   string    `json:"email_id"`
	Folder    string    `json:"folder"`
	ParentID  string    `json:"parent_id"`
	ThreadID  string    `json:"thread_id"`
	Flags     []string  `json:"flags"`
	BCC       []string  `json:"bcc"`
	CC        []string  `json:"cc"`
	Date      time.Time `json:"date"`
	From      []string  `json:"from"`
	InReplyTo []string  `json:"in_reply_to"`
	MessageID string    `json:"message_id"`
	// ReplyTo is the API's capitalised "ReplyTo" key.
	ReplyTo      []string  `json:"ReplyTo"`
	To           []string  `json:"to"`
	Subject      string    `json:"subject"`
	Size         int64     `json:"size"`
	InternalDate time.Time `json:"internal_date"`
	ModSeq       int64     `json:"mod_seq"`
	BodyPlain    string    `json:"body_plain"`
	// BodyHTML is sanitized and safe to render.
	BodyHTML      string `json:"body_html"`
	BodyTruncated bool   `json:"body_truncated"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the message and keeps the raw object in Raw.
func (v *Message) UnmarshalJSON(b []byte) error {
	type alias Message
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// ThreadListParams filters GET /unibox. The default excludes spam, trash and archive.
type ThreadListParams struct {
	Cursor string `url:"cursor"`
	// Limit is 1 to 100; the default is 50.
	Limit   int    `url:"limit"`
	From    string `url:"from"`
	Address string `url:"address"`
	// Direction is sent or received.
	Direction string `url:"direction"`
	// Folder is inbox, sent, drafts, archive, spam or trash.
	Folder          string `url:"folder"`
	IncludeArchived bool   `url:"include_archived"`
	Subject         string `url:"subject"`
	Unseen          bool   `url:"unseen"`
	// Snoozed is "true" or "any".
	Snoozed       string `url:"snoozed"`
	AwaitingReply bool   `url:"awaiting_reply"`
	AgentDrafts   bool   `url:"agent_drafts"`
	// Since and Until bound the time window, sent as RFC 3339.
	Since         time.Time `url:"since"`
	Until         time.Time `url:"until"`
	CampaignID    string    `url:"campaign_id"`
	ContactID     string    `url:"contact_id"`
	EmailID       string    `url:"email_id"`
	EmailIDs      []string  `url:"email_ids"`
	Uncategorized bool      `url:"uncategorized"`
	Automated     *bool     `url:"automated"`
	CategoryIDs   []string  `url:"category_ids"`
}

// ThreadParams selects one thread for GET /unibox/thread.
type ThreadParams struct {
	// ThreadID is the provider thread id. Required.
	ThreadID string `url:"thread_id"`
	// EmailID limits the read to one mailbox; empty reads across every mailbox.
	EmailID string `url:"email_id"`
	// Limit is 1 to 500; the default is 500.
	Limit  int    `url:"limit"`
	Cursor string `url:"cursor"`
}

// ReplyParams is the body of POST /unibox/reply.
type ReplyParams struct {
	// EmailAccountID is the sending mailbox. Required.
	EmailAccountID string `json:"email_account_id"`
	// To needs at least one address. Required.
	To []string `json:"to"`
	// Subject is required.
	Subject  string   `json:"subject"`
	CC       []string `json:"cc,omitempty"`
	BCC      []string `json:"bcc,omitempty"`
	BodyHTML string   `json:"body_html,omitempty"`
	// BodyPlain is rendered from BodyHTML when empty.
	BodyPlain string   `json:"body_plain,omitempty"`
	InReplyTo []string `json:"in_reply_to,omitempty"`
	ThreadID  string   `json:"thread_id,omitempty"`
	// SendMode is instant (default), smart or scheduled.
	SendMode string `json:"send_mode,omitempty"`
	// ScheduledAt implies scheduled.
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	// ForwardMessageID makes the call a forward.
	ForwardMessageID string `json:"forward_message_id,omitempty"`
}

// SendResult is the answer to a call that queues mail.
type SendResult struct {
	TaskID      string    `json:"task_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	SendMode    string    `json:"send_mode"`
}

// ComposeDraft is a saved compose draft.
type ComposeDraft struct {
	ID             string    `json:"id"`
	EmailAccountID string    `json:"email_account_id,omitempty"`
	To             []string  `json:"to"`
	CC             []string  `json:"cc"`
	BCC            []string  `json:"bcc"`
	Subject        string    `json:"subject"`
	Body           string    `json:"body"`
	BodyHTML       string    `json:"body_html"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DraftParams is the body of PUT /unibox/drafts/{id}.
type DraftParams struct {
	EmailAccountID string   `json:"email_account_id,omitempty"`
	To             []string `json:"to,omitempty"`
	CC             []string `json:"cc,omitempty"`
	BCC            []string `json:"bcc,omitempty"`
	Subject        string   `json:"subject,omitempty"`
	// Body is plain text, up to 100,000 characters.
	Body     string `json:"body,omitempty"`
	BodyHTML string `json:"body_html,omitempty"`
}

// AgentDraft is a reply the assistant drafted for review.
type AgentDraft struct {
	ID              string  `json:"id"`
	OrganizationID  string  `json:"organization_id"`
	EmailAccountID  string  `json:"email_account_id"`
	OwnerUserID     string  `json:"owner_user_id"`
	ThreadID        string  `json:"thread_id"`
	SourceMessageID string  `json:"source_message_id,omitempty"`
	ContactID       string  `json:"contact_id,omitempty"`
	CampaignID      string  `json:"campaign_id,omitempty"`
	ToAddr          string  `json:"to_addr"`
	Subject         string  `json:"subject"`
	InReplyTo       string  `json:"in_reply_to"`
	Body            string  `json:"body"`
	IntentClass     string  `json:"intent_class"`
	Confidence      float64 `json:"confidence"`
	Model           string  `json:"model"`
	Status          string  `json:"status"`
	// Grounding holds the draft check, including "flags". Kept untyped.
	Grounding map[string]any `json:"grounding,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Raw holds the full object, including fields this type does not name.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the draft and keeps the raw object in Raw.
func (v *AgentDraft) UnmarshalJSON(b []byte) error {
	type alias AgentDraft
	return unmarshalKeepRaw(b, (*alias)(v), &v.Raw)
}

// ApproveDraftParams is the optional body of an agent draft approval.
type ApproveDraftParams struct {
	// Body replaces the drafted text.
	Body string `json:"body,omitempty"`
	// AcknowledgeFlags is required when the draft's grounding flags are not empty.
	AcknowledgeFlags bool `json:"acknowledge_flags,omitempty"`
}

// UniboxService reads and answers the shared inbox.
type UniboxService struct{ c *Client }

// ListThreads returns one page of conversations, newest message first
// (GET /unibox). Scope READ_UNIBOX.
func (s *UniboxService) ListThreads(ctx context.Context, params *ThreadListParams, opts ...RequestOption) (*Page[MessagePreview], error) {
	var out Page[MessagePreview]
	if _, err := s.c.Request(ctx, "GET", "/unibox", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListThreadsAutoPaging walks every conversation matching params.
func (s *UniboxService) ListThreadsAutoPaging(params *ThreadListParams, opts ...RequestOption) *Pager[MessagePreview] {
	return Paginate[MessagePreview](s.c, "GET", "/unibox", encodeQuery(params), nil, opts...)
}

// GetThread returns one page of a thread's messages as previews
// (GET /unibox/thread). Read bodies with GetMessage. Scope READ_UNIBOX.
func (s *UniboxService) GetThread(ctx context.Context, params *ThreadParams, opts ...RequestOption) (*Page[MessagePreview], error) {
	var out Page[MessagePreview]
	if _, err := s.c.Request(ctx, "GET", "/unibox/thread", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetThreadAutoPaging walks every message of a thread.
func (s *UniboxService) GetThreadAutoPaging(params *ThreadParams, opts ...RequestOption) *Pager[MessagePreview] {
	return Paginate[MessagePreview](s.c, "GET", "/unibox/thread", encodeQuery(params), nil, opts...)
}

// GetMessage returns one full message (GET /unibox/{id}). Reading marks it
// read for an unrestricted caller. Scope READ_UNIBOX.
func (s *UniboxService) GetMessage(ctx context.Context, id string, opts ...RequestOption) (*Message, error) {
	var out Message
	if _, err := s.c.Request(ctx, "GET", pathf("/unibox/%s", id), nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// UnreadCount returns the number of unread conversations (GET /unibox/count),
// for one mailbox when emailID is not empty. Scope READ_UNIBOX.
func (s *UniboxService) UnreadCount(ctx context.Context, emailID string, opts ...RequestOption) (int, error) {
	var out struct {
		Count int `json:"count"`
	}
	q := encodeQuery(&struct {
		EmailID string `url:"email_id"`
	}{emailID})
	if _, err := s.c.Request(ctx, "GET", "/unibox/count", q, nil, &out, opts...); err != nil {
		return 0, err
	}
	return out.Count, nil
}

// Reply sends a reply, a new message in a thread or a forward
// (POST /unibox/reply). It sends real mail. Scope WRITE_UNIBOX.
func (s *UniboxService) Reply(ctx context.Context, params *ReplyParams, opts ...RequestOption) (*SendResult, error) {
	var out SendResult
	if _, err := s.c.Request(ctx, "POST", "/unibox/reply", nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDrafts returns the caller's compose drafts (GET /unibox/drafts). Scope READ_UNIBOX.
func (s *UniboxService) ListDrafts(ctx context.Context, opts ...RequestOption) ([]ComposeDraft, error) {
	var out struct {
		Data []ComposeDraft `json:"data"`
	}
	if _, err := s.c.Request(ctx, "GET", "/unibox/drafts", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// SaveDraft creates or replaces a compose draft under a client-generated
// uuid (PUT /unibox/drafts/{id}) and returns its id. It never sends. Scope WRITE_UNIBOX.
func (s *UniboxService) SaveDraft(ctx context.Context, id string, params *DraftParams, opts ...RequestOption) (string, error) {
	if params == nil {
		params = &DraftParams{}
	}
	var out struct {
		ID string `json:"id"`
	}
	if _, err := s.c.Request(ctx, "PUT", pathf("/unibox/drafts/%s", id), nil, params, &out, opts...); err != nil {
		return "", err
	}
	return out.ID, nil
}

// DeleteDraft deletes a compose draft (DELETE /unibox/drafts/{id}). Deleting
// a missing draft is a no-op. Scope WRITE_UNIBOX.
func (s *UniboxService) DeleteDraft(ctx context.Context, id string, opts ...RequestOption) error {
	_, err := s.c.Request(ctx, "DELETE", pathf("/unibox/drafts/%s", id), nil, nil, nil, opts...)
	return err
}

// ListAgentDrafts returns the assistant's drafts awaiting review
// (GET /unibox/agent-drafts). Scope READ_UNIBOX.
func (s *UniboxService) ListAgentDrafts(ctx context.Context, opts ...RequestOption) ([]AgentDraft, error) {
	var out struct {
		Data []AgentDraft `json:"data"`
	}
	if _, err := s.c.Request(ctx, "GET", "/unibox/agent-drafts", nil, nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ApproveAgentDraft sends an agent draft (POST /unibox/agent-drafts/{id}/approve).
// It sends real mail. params may be nil. A draft with grounding flags answers
// 409 draft_flagged until AcknowledgeFlags is set. Scope WRITE_UNIBOX.
func (s *UniboxService) ApproveAgentDraft(ctx context.Context, id string, params *ApproveDraftParams, opts ...RequestOption) (*SendResult, error) {
	var out SendResult
	if _, err := s.c.Request(ctx, "POST", pathf("/unibox/agent-drafts/%s/approve", id), nil, params, &out, opts...); err != nil {
		return nil, err
	}
	return &out, nil
}

// DiscardAgentDraft discards an agent draft (POST /unibox/agent-drafts/{id}/discard)
// and returns its new status. Scope WRITE_UNIBOX.
func (s *UniboxService) DiscardAgentDraft(ctx context.Context, id string, opts ...RequestOption) (string, error) {
	var out struct {
		Status string `json:"status"`
	}
	if _, err := s.c.Request(ctx, "POST", pathf("/unibox/agent-drafts/%s/discard", id), nil, nil, &out, opts...); err != nil {
		return "", err
	}
	return out.Status, nil
}
