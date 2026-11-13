// Package ayinbox is the official Go client for the AY Inbox REST API.
//
// AY Inbox is a hosted cold email platform. This package is a client of its
// public API at https://api.ayinbox.com/v1. Create a client with NewClient,
// then call the resource services hanging off it:
//
//	client, err := ayinbox.NewClient() // reads AYINBOX_API_KEY
//	if err != nil {
//		log.Fatal(err)
//	}
//	me, err := client.Me.Get(ctx)
//
// Every call takes a context.Context first and returns an *Error for any
// non-2xx answer or network failure. Use errors.Is with the sentinel errors
// (ErrNotFound, ErrRateLimited, ...) or the Is* predicates to branch.
package ayinbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Version is the version of this SDK. It is sent in the User-Agent header.
const Version = "0.1.0"

// DefaultBaseURL is the AY Inbox API base URL used when none is configured.
const DefaultBaseURL = "https://api.ayinbox.com/v1"

// DefaultTimeout is the per-attempt timeout used when none is configured.
const DefaultTimeout = 30 * time.Second

// DefaultMaxRetries is the number of automatic retries used when none is configured.
const DefaultMaxRetries = 2

const (
	userAgent        = "ayinbox-go/" + Version
	maxRetryAfter    = 120 * time.Second
	backoffBase      = 500 * time.Millisecond
	backoffCap       = 8 * time.Second
	idempotencyHdr   = "Idempotency-Key"
	requestIDHdr     = "X-Request-Id"
	envAPIKey        = "AYINBOX_API_KEY"
	envBaseURL       = "AYINBOX_BASE_URL"
	maxErrorBodySize = 1 << 20
)

// ErrMissingAPIKey is returned by NewClient when no API key was passed and
// AYINBOX_API_KEY is empty.
var ErrMissingAPIKey = errors.New("ayinbox: no API key: pass WithAPIKey or set AYINBOX_API_KEY")

// Client is an AY Inbox API client. It is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	sleep      func(context.Context, time.Duration) error

	// Me reads the identity behind the credential.
	Me *MeService
	// Campaigns manages campaigns, their steps and their leads.
	Campaigns *CampaignsService
	// Contacts manages contacts.
	Contacts *ContactsService
	// Mailboxes reads connected mailboxes (the /emails routes).
	Mailboxes *MailboxesService
	// Unibox reads and answers the shared inbox.
	Unibox *UniboxService
	// Templates manages reply templates.
	Templates *TemplatesService
	// Webhooks manages webhook endpoints and deliveries.
	Webhooks *WebhooksService
	// Analytics reads campaign and mailbox analytics.
	Analytics *AnalyticsService
	// Suppressions manages the suppression list.
	Suppressions *SuppressionsService
	// AITools lists and calls the agent tools exposed over HTTP.
	AITools *AIToolsService
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key or OAuth access token. Without it the client
// reads AYINBOX_API_KEY.
func WithAPIKey(key string) Option { return func(c *Client) { c.apiKey = key } }

// WithBaseURL overrides the API base URL. Without it the client reads
// AYINBOX_BASE_URL, then uses DefaultBaseURL. A trailing slash is removed.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithHTTPClient sets the *http.Client used to send requests.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }

// WithMaxRetries sets how many times a retryable request is retried.
// 0 disables retries. The default is 2.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
	}
}

// WithTimeout sets the timeout of each attempt. The default is 30 seconds.
// A value of 0 or less disables the per-attempt timeout.
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// NewClient returns a client configured by opts and the AYINBOX_API_KEY and
// AYINBOX_BASE_URL environment variables. It returns ErrMissingAPIKey when no
// key is available.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
		sleep:      sleepCtx,
	}
	for _, o := range opts {
		o(c)
	}
	if c.apiKey == "" {
		c.apiKey = os.Getenv(envAPIKey)
	}
	if c.apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	if c.baseURL == "" {
		c.baseURL = os.Getenv(envBaseURL)
	}
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	c.baseURL = strings.TrimRight(c.baseURL, "/")
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}
	c.Me = &MeService{c: c}
	c.Campaigns = &CampaignsService{c: c, Steps: &CampaignStepsService{c: c}, Leads: &CampaignLeadsService{c: c}}
	c.Contacts = &ContactsService{c: c}
	c.Mailboxes = &MailboxesService{c: c}
	c.Unibox = &UniboxService{c: c}
	c.Templates = &TemplatesService{c: c}
	c.Webhooks = &WebhooksService{c: c}
	c.Analytics = &AnalyticsService{c: c}
	c.Suppressions = &SuppressionsService{c: c}
	c.AITools = &AIToolsService{c: c}
	return c, nil
}

// BaseURL returns the base URL the client sends requests to.
func (c *Client) BaseURL() string { return c.baseURL }

// String describes the client without its credential.
func (c *Client) String() string {
	return fmt.Sprintf("ayinbox.Client{baseURL: %q, apiKey: %s}", c.baseURL, redact(c.apiKey))
}

// GoString describes the client without its credential.
func (c *Client) GoString() string { return c.String() }

func redact(key string) string {
	if key == "" {
		return `""`
	}
	return `"[redacted]"`
}

// Response is the raw HTTP answer to a request.
type Response struct {
	// StatusCode is the HTTP status.
	StatusCode int
	// Header holds the response headers, including X-Request-Id and the
	// X-RateLimit-* headers.
	Header http.Header
	// Body is the raw response body.
	Body []byte
}

// RequestID returns the X-Request-Id response header.
func (r *Response) RequestID() string { return r.Header.Get(requestIDHdr) }

// RequestOption configures a single call.
type RequestOption func(*requestConfig)

type requestConfig struct {
	idempotencyKey string
	timeout        *time.Duration
	header         http.Header
	capture        *Response
	retrySafe      bool
}

// WithIdempotencyKey sends key as the Idempotency-Key header. A POST or PATCH
// is retried automatically only when it carries a key (or on 429, which never
// runs the request). Use NewIdempotencyKey to make one.
func WithIdempotencyKey(key string) RequestOption {
	return func(rc *requestConfig) { rc.idempotencyKey = key }
}

// WithRequestTimeout overrides the client's per-attempt timeout for one call.
func WithRequestTimeout(d time.Duration) RequestOption {
	return func(rc *requestConfig) { rc.timeout = &d }
}

// WithHeader adds a header to one call. It cannot replace Authorization.
func WithHeader(key, value string) RequestOption {
	return func(rc *requestConfig) {
		if rc.header == nil {
			rc.header = http.Header{}
		}
		rc.header.Set(key, value)
	}
}

// WithRequestID sends id as the X-Request-Id header. The API echoes it in the
// response and in error request ids.
func WithRequestID(id string) RequestOption { return WithHeader(requestIDHdr, id) }

// WithResponse copies the final HTTP response (status, headers, body) into r
// when the call returns, so callers can read headers such as X-RateLimit-Remaining.
func WithResponse(r *Response) RequestOption {
	return func(rc *requestConfig) { rc.capture = r }
}

// withRetrySafe marks a non-mutating POST (a search) as safe to retry.
func withRetrySafe() RequestOption { return func(rc *requestConfig) { rc.retrySafe = true } }

// NewIdempotencyKey returns a random UUIDv4 suitable for WithIdempotencyKey.
func NewIdempotencyKey() string { return newUUID() }

// Request sends any API request through the client's authentication, retries
// and error mapping. path is relative to the base URL (for example
// "/campaigns/123/send-plan"). body, when not nil, is encoded as JSON unless it
// is a []byte or json.RawMessage, which is sent as is. When out is not nil and
// the response has a body, the body is decoded into it as JSON. The raw
// response is always returned on success.
func (c *Client) Request(ctx context.Context, method, path string, query url.Values, body, out any, opts ...RequestOption) (*Response, error) {
	rc := &requestConfig{}
	for _, o := range opts {
		o(rc)
	}
	resp, err := c.do(ctx, method, path, query, body, rc)
	if rc.capture != nil && resp != nil {
		*rc.capture = *resp
	}
	if err != nil {
		return resp, err
	}
	if out != nil && len(bytes.TrimSpace(resp.Body)) > 0 {
		if err := json.Unmarshal(resp.Body, out); err != nil {
			return resp, fmt.Errorf("ayinbox: decode response of %s %s: %w", method, path, err)
		}
	}
	return resp, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, rc *requestConfig) (*Response, error) {
	method = strings.ToUpper(method)
	u := c.baseURL + "/" + strings.TrimPrefix(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	if isNilValue(body) {
		body = nil
	}
	var payload []byte
	if body != nil {
		switch b := body.(type) {
		case []byte:
			payload = b
		case json.RawMessage:
			payload = b
		default:
			var err error
			payload, err = json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("ayinbox: encode request body: %w", err)
			}
		}
	}
	safe := rc.retrySafe || rc.idempotencyKey != "" || (method != http.MethodPost && method != http.MethodPatch)

	for attempt := 0; ; attempt++ {
		resp, err := c.attempt(ctx, method, u, payload, body != nil, rc)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if attempt < c.maxRetries && safe {
				if serr := c.sleep(ctx, backoff(attempt)); serr != nil {
					return nil, serr
				}
				continue
			}
			return nil, err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		apiErr := parseError(resp)
		if attempt < c.maxRetries && retryableStatus(resp.StatusCode) && (safe || resp.StatusCode == http.StatusTooManyRequests) {
			wait := apiErr.RetryAfter
			if wait <= 0 {
				wait = backoff(attempt)
			}
			if wait > maxRetryAfter {
				wait = maxRetryAfter
			}
			if serr := c.sleep(ctx, wait); serr != nil {
				return resp, serr
			}
			continue
		}
		return resp, apiErr
	}
}

func (c *Client) attempt(ctx context.Context, method, u string, payload []byte, hasBody bool, rc *requestConfig) (*Response, error) {
	timeout := c.timeout
	if rc.timeout != nil {
		timeout = *rc.timeout
	}
	actx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		actx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	var rdr io.Reader
	if hasBody {
		rdr = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(actx, method, u, rdr)
	if err != nil {
		return nil, fmt.Errorf("ayinbox: build request: %w", err)
	}
	for k, vs := range rc.header {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if rc.idempotencyKey != "" {
		req.Header.Set(idempotencyHdr, rc.idempotencyKey)
	}
	hr, err := c.httpClient.Do(req)
	if err != nil {
		return nil, networkError(err, ctx, actx)
	}
	defer hr.Body.Close()
	data, err := io.ReadAll(hr.Body)
	if err != nil {
		return nil, networkError(err, ctx, actx)
	}
	return &Response{StatusCode: hr.StatusCode, Header: hr.Header, Body: data}, nil
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

func retryableStatus(s int) bool {
	switch s {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// backoff is exponential with full jitter: random(0, min(cap, base*2^attempt)).
func backoff(attempt int) time.Duration {
	d := backoffCap
	if attempt < 5 {
		d = min(backoffCap, backoffBase<<attempt)
	}
	return time.Duration(rand.Int64N(int64(d) + 1))
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if n, err := strconv.ParseFloat(h, 64); err == nil && n >= 0 {
		return time.Duration(n * float64(time.Second))
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func pathf(format string, ids ...string) string {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = url.PathEscape(id)
	}
	return fmt.Sprintf(format, args...)
}
