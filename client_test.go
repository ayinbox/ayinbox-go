package ayinbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeaders(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	me, err := c.Me.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Email != "dana@example.com" || me.AuthType != "api_key" || len(me.Scopes) != 3 {
		t.Fatalf("unexpected identity: %+v", me)
	}
	if len(me.Raw) == 0 {
		t.Fatal("Raw not kept")
	}
	r := fs.last()
	if r.Path != "/v1/me" {
		t.Fatalf("path %q", r.Path)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+testKey {
		t.Fatalf("Authorization %q", got)
	}
	if got := r.Header.Get("User-Agent"); got != "ayinbox-go/"+Version {
		t.Fatalf("User-Agent %q", got)
	}
	if got := r.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("Accept %q", got)
	}
	if got := r.Header.Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type on a GET: %q", got)
	}
	if r.Header.Get("Idempotency-Key") != "" {
		t.Fatal("unexpected Idempotency-Key")
	}
}

func TestNewClientEnvironment(t *testing.T) {
	t.Setenv("AYINBOX_API_KEY", "")
	t.Setenv("AYINBOX_BASE_URL", "")
	if _, err := NewClient(); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("want ErrMissingAPIKey, got %v", err)
	}
	t.Setenv("AYINBOX_API_KEY", testKey)
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != DefaultBaseURL {
		t.Fatalf("base URL %q", c.BaseURL())
	}
	t.Setenv("AYINBOX_BASE_URL", "https://example.test/v1/")
	c, _ = NewClient()
	if c.BaseURL() != "https://example.test/v1" {
		t.Fatalf("base URL %q", c.BaseURL())
	}
	c, _ = NewClient(WithBaseURL("https://other.test/v1"))
	if c.BaseURL() != "https://other.test/v1" {
		t.Fatalf("option should win over env, got %q", c.BaseURL())
	}
}

func TestKeyNeverPrinted(t *testing.T) {
	c, err := NewClient(WithAPIKey(testKey))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{fmt.Sprint(c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), c.String()} {
		if strings.Contains(s, testKey) {
			t.Fatalf("key leaked in %q", s)
		}
	}
}

func TestTypedErrors(t *testing.T) {
	cases := []struct {
		file     string
		sentinel error
		code     string
		pred     func(error) bool
	}{
		{"errors/400_bad_request.json", ErrBadRequest, "bad_request", IsBadRequest},
		{"errors/400_invalid_cursor.json", ErrValidation, "bad_request", IsBadRequest},
		{"errors/401_unauthorized.json", ErrAuthentication, "unauthorized", IsAuthentication},
		{"errors/401_api_key_holder_left.json", ErrAuthentication, "api_key_holder_left", IsAuthentication},
		{"errors/402_insufficient_credits.json", ErrPaymentRequired, "insufficient_credits", IsPaymentRequired},
		{"errors/403_forbidden.json", ErrPermissionDenied, "forbidden", IsPermissionDenied},
		{"errors/403_api_key_mailbox_limited.json", ErrPermissionDenied, "api_key_mailbox_limited", IsPermissionDenied},
		{"errors/404_not_found.json", ErrNotFound, "not_found", IsNotFound},
		{"errors/409_conflict.json", ErrConflict, "conflict", IsConflict},
		{"errors/409_idempotency_outcome_unknown.json", ErrConflict, "idempotency_outcome_unknown", IsConflict},
		{"errors/409_idempotent_secret_withheld.json", ErrConflict, "idempotent_secret_withheld", IsConflict},
		{"errors/422_unprocessable.json", ErrUnprocessable, "unprocessable", IsBadRequest},
		{"errors/429_api_key_rate_limited.json", ErrRateLimited, "rate_limit_exceeded", IsRateLimited},
		{"errors/429_category_rate_limited.json", ErrRateLimited, "rate_limit_exceeded", IsRateLimited},
		{"errors/500_internal_error.json", ErrInternalServer, "internal_error", IsServerError},
		{"errors/503_service_unavailable.json", ErrServiceUnavailable, "service_unavailable", IsServerError},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			fs := newFixtureServer(t)
			fs.handle("GET", "/v1/me", func(w http.ResponseWriter, _ *http.Request) {
				writeErrorFixture(t, w, tc.file, fs.idx)
			})
			c, _ := newTestClient(t, fs, WithMaxRetries(0))
			_, err := c.Me.Get(context.Background())
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("want *Error, got %T %v", err, err)
			}
			if !errors.Is(err, tc.sentinel) || !tc.pred(err) {
				t.Fatalf("%v does not match its category", err)
			}
			if e.Code != tc.code || ErrorCode(err) != tc.code {
				t.Fatalf("code %q, want %q", e.Code, tc.code)
			}
			if e.Status != fs.idx.Errors[tc.file].Status || e.Message == "" || len(e.Body) == 0 {
				t.Fatalf("incomplete error: %+v", e)
			}
			if !strings.Contains(tc.file, "category") && e.RequestID != "req-fixture-0001" {
				t.Fatalf("request id %q", e.RequestID)
			}
			if errors.Is(err, ErrConnection) || strings.Contains(err.Error(), testKey) {
				t.Fatalf("bad error: %v", err)
			}
		})
	}
}

func TestRateLimitWithoutCode(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("GET", "/v1/templates", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		_, _ = w.Write(readFixture(t, "errors/429_category_rate_limited.json"))
	})
	c, _ := newTestClient(t, fs, WithMaxRetries(0))
	_, err := c.Templates.List(context.Background(), "")
	var e *Error
	if !errors.As(err, &e) || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want rate limit error, got %v", err)
	}
	if e.Code != "rate_limit_exceeded" || e.RetryAfter != 12*time.Second || e.RequestID != "" {
		t.Fatalf("parsed %+v", e)
	}
	if e.Message != "Rate limit exceeded for write operations" {
		t.Fatalf("message %q", e.Message)
	}
}

func TestRateLimitBodyRetryAfterAndDefaults(t *testing.T) {
	r := &Response{StatusCode: 429, Header: http.Header{"X-Request-Id": {"hdr-id"}}, Body: []byte(`{"error":"rate_limit_exceeded","message":"x","retry_after":7}`)}
	e := parseError(r)
	if e.RetryAfter != 7*time.Second || e.Code != "rate_limit_exceeded" || e.RequestID != "hdr-id" {
		t.Fatalf("%+v", e)
	}
	e = parseError(&Response{StatusCode: 502, Header: http.Header{}, Body: []byte("<html>bad gateway</html>")})
	if e.Code != "internal_error" || e.Message != "<html>bad gateway</html>" || !errors.Is(e, ErrInternalServer) {
		t.Fatalf("%+v", e)
	}
	e = parseError(&Response{StatusCode: 403, Header: http.Header{}, Body: []byte(`{"error":"Forbidden","message":"no"}`)})
	if e.Code != "forbidden" {
		t.Fatalf("code %q", e.Code)
	}
}

func TestRetry503ThenSuccess(t *testing.T) {
	fs := newFixtureServer(t)
	var n atomic.Int32
	fs.handle("GET", "/v1/me", func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			writeErrorFixture(t, w, "errors/503_service_unavailable.json", fs.idx)
			return
		}
		_, _ = w.Write(readFixture(t, "me.json"))
	})
	c, rec := newTestClient(t, fs)
	if _, err := c.Me.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 2 || len(rec.waits) != 1 || rec.waits[0] != 5*time.Second {
		t.Fatalf("attempts %d waits %v", n.Load(), rec.waits)
	}
}

func TestRetry429RetryAfterThenSuccess(t *testing.T) {
	fs := newFixtureServer(t)
	var n atomic.Int32
	fs.handle("GET", "/v1/campaigns", func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			writeErrorFixture(t, w, "errors/429_api_key_rate_limited.json", fs.idx)
			return
		}
		_, _ = w.Write(readFixture(t, "campaigns_list_page2.json"))
	})
	c, rec := newTestClient(t, fs)
	page, err := c.Campaigns.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || n.Load() != 2 || len(rec.waits) != 1 || rec.waits[0] != 12*time.Second {
		t.Fatalf("attempts %d waits %v", n.Load(), rec.waits)
	}
}

func TestRetriesExhausted(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("GET", "/v1/me", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(502)
	})
	c, rec := newTestClient(t, fs, WithMaxRetries(2))
	_, err := c.Me.Get(context.Background())
	if !errors.Is(err, ErrServer) || fs.count() != 3 || len(rec.waits) != 2 {
		t.Fatalf("err %v attempts %d waits %v", err, fs.count(), rec.waits)
	}
	for _, w := range rec.waits {
		if w < 0 || w > backoffCap {
			t.Fatalf("backoff %v out of range", w)
		}
	}
}

func TestNoRetryOfKeylessPost(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("POST", "/v1/campaigns", func(w http.ResponseWriter, _ *http.Request) {
		writeErrorFixture(t, w, "errors/503_service_unavailable.json", fs.idx)
	})
	c, _ := newTestClient(t, fs)
	_, err := c.Campaigns.Create(context.Background(), &CampaignCreateParams{Name: "x"})
	if !errors.Is(err, ErrServiceUnavailable) || fs.count() != 1 {
		t.Fatalf("err %v attempts %d", err, fs.count())
	}
}

func TestKeyedPostRetriedWithSameKey(t *testing.T) {
	fs := newFixtureServer(t)
	var n atomic.Int32
	fs.handle("POST", "/v1/campaigns", func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			writeErrorFixture(t, w, "errors/503_service_unavailable.json", fs.idx)
			return
		}
		_, _ = w.Write(readFixture(t, "campaign.json"))
	})
	c, _ := newTestClient(t, fs)
	if _, err := c.Campaigns.Create(context.Background(), &CampaignCreateParams{Name: "x"}, WithIdempotencyKey("key-1")); err != nil {
		t.Fatal(err)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.reqs) != 2 || fs.reqs[0].Header.Get("Idempotency-Key") != "key-1" || fs.reqs[1].Header.Get("Idempotency-Key") != "key-1" {
		t.Fatalf("requests %d", len(fs.reqs))
	}
}

func TestKeylessPostRetriedOn429(t *testing.T) {
	fs := newFixtureServer(t)
	var n atomic.Int32
	fs.handle("POST", "/v1/campaigns", func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			writeErrorFixture(t, w, "errors/429_api_key_rate_limited.json", fs.idx)
			return
		}
		_, _ = w.Write(readFixture(t, "campaign.json"))
	})
	c, _ := newTestClient(t, fs)
	if _, err := c.Campaigns.Create(context.Background(), &CampaignCreateParams{Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 2 {
		t.Fatalf("attempts %d", n.Load())
	}
}

func TestContactSearchRetriedWithoutKey(t *testing.T) {
	fs := newFixtureServer(t)
	var n atomic.Int32
	fs.handle("POST", "/v1/contacts/search", func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) == 1 {
			writeErrorFixture(t, w, "errors/503_service_unavailable.json", fs.idx)
			return
		}
		_, _ = w.Write(readFixture(t, "contacts_list.json"))
	})
	c, _ := newTestClient(t, fs)
	if _, err := c.Contacts.List(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 2 {
		t.Fatalf("attempts %d", n.Load())
	}
}

func TestNoRetryOn500Or409(t *testing.T) {
	for _, f := range []string{"errors/500_internal_error.json", "errors/409_idempotency_outcome_unknown.json"} {
		fs := newFixtureServer(t)
		fs.handle("DELETE", "/v1/templates/abc", func(w http.ResponseWriter, _ *http.Request) {
			writeErrorFixture(t, w, f, fs.idx)
		})
		c, _ := newTestClient(t, fs)
		if err := c.Templates.Delete(context.Background(), "abc", WithIdempotencyKey("k")); err == nil || fs.count() != 1 {
			t.Fatalf("%s: err %v attempts %d", f, err, fs.count())
		}
	}
}

func TestIdempotencyAndExtraHeaders(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	key := NewIdempotencyKey()
	if len(key) != 36 || key[14] != '4' {
		t.Fatalf("bad uuid %q", key)
	}
	var resp Response
	_, err := c.Contacts.Create(context.Background(), []ContactInput{{Email: "lee@example.com"}},
		WithIdempotencyKey(key), WithRequestID("my-req-1"), WithHeader("X-Extra", "1"), WithResponse(&resp))
	if err != nil {
		t.Fatal(err)
	}
	r := fs.last()
	if r.Header.Get("Idempotency-Key") != key || r.Header.Get("X-Request-Id") != "my-req-1" || r.Header.Get("X-Extra") != "1" {
		t.Fatalf("headers %v", r.Header)
	}
	if r.Header.Get("Content-Type") != "application/json" {
		t.Fatal("missing Content-Type")
	}
	if resp.StatusCode != 200 || resp.Header.Get("X-RateLimit-Remaining") != "59" || resp.RequestID() != "req-fixture-0001" {
		t.Fatalf("captured %+v", resp)
	}
}

func TestPaginationWalk(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()
	pager := c.Campaigns.ListAutoPaging(nil)
	var ids []string
	for pager.Next(ctx) {
		ids = append(ids, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || fs.count() != 2 {
		t.Fatalf("ids %v requests %d", ids, fs.count())
	}
	if q := fs.last().Query; q != "cursor=c1_ZmFrZS1jdXJzb3ItcGFnZS0y" {
		t.Fatalf("second page query %q", q)
	}
	if pager.Next(ctx) {
		t.Fatal("pager should stay done")
	}

	all, err := c.Campaigns.ListAutoPaging(nil).Collect(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("collect %d %v", len(all), err)
	}
}

func TestPaginationResendsSearchBody(t *testing.T) {
	fs := newFixtureServer(t)
	var bodies []string
	fs.handle("POST", "/v1/contacts/search", func(w http.ResponseWriter, r *http.Request) {
		b := fs.last().Body
		bodies = append(bodies, string(b))
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"data":[{"id":"a"}],"pagination":{"next_cursor":"t1_next","has_more":true,"total":2},"counts":{"all":2}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"b"}],"pagination":{"next_cursor":null,"has_more":false,"total":2}}`))
	})
	c, _ := newTestClient(t, fs)
	all, err := c.Contacts.ListAutoPaging(&ContactListParams{Limit: 1, Search: ContactSearch{Query: "lee", SortBy: "email"}}).Collect(context.Background())
	if err != nil || len(all) != 2 || all[1].ID != "b" {
		t.Fatalf("%v %v", all, err)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] || !strings.Contains(bodies[0], `"sort_by":"email"`) {
		t.Fatalf("bodies %v", bodies)
	}
	if q := fs.last().Query; !strings.Contains(q, "cursor=t1_next") || !strings.Contains(q, "limit=1") {
		t.Fatalf("query %q", q)
	}
}

func TestPaginationErrorStopsWalk(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("GET", "/v1/suppressions", func(w http.ResponseWriter, _ *http.Request) {
		writeErrorFixture(t, w, "errors/400_invalid_cursor.json", fs.idx)
	})
	c, _ := newTestClient(t, fs)
	_, err := c.Suppressions.ListAutoPaging(nil).Collect(context.Background())
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err %v", err)
	}
}

func TestGenericRequestAndPaginate(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("GET", "/v1/campaigns/abc/send-plan", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"days":[1,2]}`))
	})
	c, _ := newTestClient(t, fs)
	var out map[string]any
	resp, err := c.Request(context.Background(), "get", "campaigns/abc/send-plan", nil, nil, &out)
	if err != nil || resp.StatusCode != 200 || out["days"] == nil {
		t.Fatalf("%v %v", out, err)
	}
	rows, err := Paginate[map[string]any](c, "GET", "/campaigns", nil, nil).Collect(context.Background())
	if err != nil || len(rows) != 3 {
		t.Fatalf("%d %v", len(rows), err)
	}
	var raw json.RawMessage
	if _, err := c.Request(context.Background(), "POST", "/campaigns", nil, json.RawMessage(`{"name":"raw"}`), &raw); err != nil {
		t.Fatal(err)
	}
	if string(fs.last().Body) != `{"name":"raw"}` {
		t.Fatalf("body %s", fs.last().Body)
	}
}

func TestConnectionErrorAndTimeout(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("GET", "/v1/me", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	c, rec := newTestClient(t, fs, WithTimeout(20*time.Millisecond), WithMaxRetries(1))
	_, err := c.Me.Get(context.Background())
	if !errors.Is(err, ErrTimeout) || !errors.Is(err, ErrConnection) || !IsConnection(err) {
		t.Fatalf("want timeout, got %v", err)
	}
	if fs.count() != 2 || len(rec.waits) != 1 {
		t.Fatalf("attempts %d", fs.count())
	}

	dead, _ := NewClient(WithAPIKey(testKey), WithBaseURL("http://127.0.0.1:1/v1"), WithMaxRetries(0))
	_, err = dead.Me.Get(context.Background())
	if !errors.Is(err, ErrConnection) || errors.Is(err, ErrTimeout) {
		t.Fatalf("want connection error, got %v", err)
	}
	if strings.Contains(err.Error(), testKey) {
		t.Fatal("key in error")
	}
}

func TestContextCancelled(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Me.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestBackoffBounds(t *testing.T) {
	for attempt := 0; attempt < 10; attempt++ {
		for i := 0; i < 50; i++ {
			d := backoff(attempt)
			limit := backoffCap
			if attempt < 4 {
				limit = backoffBase << attempt
			}
			if d < 0 || d > limit {
				t.Fatalf("attempt %d: %v > %v", attempt, d, limit)
			}
		}
	}
	if parseRetryAfter("3") != 3*time.Second || parseRetryAfter("") != 0 || parseRetryAfter("soon") != 0 {
		t.Fatal("parseRetryAfter")
	}
}
