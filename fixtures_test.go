package ayinbox

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const testKey = "aybox_test_0000000000000000000000000000000000000"

type fixtureRoute struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

type fixtureIndex struct {
	DefaultSuccessHeaders map[string]string `json:"default_success_headers"`
	Routes                []fixtureRoute    `json:"routes"`
	Errors                map[string]struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
	} `json:"errors"`
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func loadIndex(t *testing.T) fixtureIndex {
	t.Helper()
	var idx fixtureIndex
	if err := json.Unmarshal(readFixture(t, "index.json"), &idx); err != nil {
		t.Fatal(err)
	}
	return idx
}

// resolveBody turns an index.json body reference into bytes. A nil result means no body.
func resolveBody(t *testing.T, raw json.RawMessage) []byte {
	t.Helper()
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var ref string
	if err := json.Unmarshal(raw, &ref); err != nil {
		return raw
	}
	if ref == "" {
		return []byte{}
	}
	switch {
	case strings.Contains(ref, "#"):
		file, ptr, _ := strings.Cut(ref, "#")
		var obj map[string][]json.RawMessage
		if err := json.Unmarshal(readFixture(t, file), &obj); err != nil {
			t.Fatal(err)
		}
		key := strings.TrimSuffix(ptr, "[0]")
		return obj[key][0]
	case strings.HasSuffix(ref, "[0]"):
		var arr []json.RawMessage
		if err := json.Unmarshal(readFixture(t, strings.TrimSuffix(ref, "[0]")), &arr); err != nil {
			t.Fatal(err)
		}
		return arr[0]
	}
	return readFixture(t, ref)
}

type recorded struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

// fixtureServer serves index.json routes, plus extra handlers registered by tests.
type fixtureServer struct {
	t       *testing.T
	srv     *httptest.Server
	idx     fixtureIndex
	mu      sync.Mutex
	reqs    []recorded
	extra   map[string]http.HandlerFunc
	headers map[string]string
}

func newFixtureServer(t *testing.T) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{t: t, idx: loadIndex(t), extra: map[string]http.HandlerFunc{}}
	fs.headers = fs.idx.DefaultSuccessHeaders
	fs.srv = httptest.NewServer(http.HandlerFunc(fs.serve))
	t.Cleanup(fs.srv.Close)
	return fs
}

func (fs *fixtureServer) handle(method, path string, h http.HandlerFunc) {
	fs.extra[method+" "+path] = h
}

func (fs *fixtureServer) last() recorded {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.reqs) == 0 {
		fs.t.Fatal("no request recorded")
	}
	return fs.reqs[len(fs.reqs)-1]
}

func (fs *fixtureServer) count() int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return len(fs.reqs)
}

func (fs *fixtureServer) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	fs.mu.Lock()
	fs.reqs = append(fs.reqs, recorded{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Clone(), body})
	fs.mu.Unlock()

	if h, ok := fs.extra[r.Method+" "+r.URL.Path]; ok {
		h(w, r)
		return
	}
	route, ok := fs.match(r, body)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(readFixture(fs.t, "errors/404_not_found.json"))
		return
	}
	for k, v := range fs.headers {
		w.Header().Set(k, v)
	}
	b := resolveBody(fs.t, route.Body)
	w.WriteHeader(route.Status)
	if b != nil {
		_, _ = w.Write(b)
	}
}

func (fs *fixtureServer) match(r *http.Request, body []byte) (fixtureRoute, bool) {
	best, bestScore := fixtureRoute{}, -1
	for _, rt := range fs.idx.Routes {
		if rt.Method != r.Method {
			continue
		}
		p, query, _ := strings.Cut(rt.Path, "?")
		p, note, _ := strings.Cut(p, " ")
		if query != "" && query != r.URL.RawQuery {
			continue
		}
		if query == "" && r.URL.Query().Get("cursor") != "" && r.URL.Path == "/v1/campaigns" {
			continue
		}
		if note != "" && !bytes.Contains(body, []byte(`"campaign_ids"`)) {
			continue
		}
		if note == "" && rt.Path == "/v1/contacts/search" && bytes.Contains(body, []byte(`"campaign_ids"`)) {
			continue
		}
		score, ok := matchPath(p, r.URL.Path)
		if !ok {
			continue
		}
		if query != "" || note != "" {
			score += 100
		}
		if score > bestScore {
			best, bestScore = rt, score
		}
	}
	return best, bestScore >= 0
}

func matchPath(pattern, path string) (int, bool) {
	ps, xs := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(ps) != len(xs) {
		return 0, false
	}
	score := 0
	for i := range ps {
		if strings.HasPrefix(ps[i], "{") {
			continue
		}
		if ps[i] != xs[i] {
			return 0, false
		}
		score++
	}
	return score, true
}

// writeErrorFixture answers with an error fixture and its listed status and headers.
func writeErrorFixture(t *testing.T, w http.ResponseWriter, name string, idx fixtureIndex) {
	e, ok := idx.Errors[name]
	if !ok {
		t.Fatalf("unknown error fixture %s", name)
	}
	for k, v := range e.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(e.Status)
	_, _ = w.Write(readFixture(t, name))
}

type sleepRecorder struct {
	mu    sync.Mutex
	waits []time.Duration
}

func (s *sleepRecorder) sleep(_ context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waits = append(s.waits, d)
	return nil
}

func newTestClient(t *testing.T, fs *fixtureServer, opts ...Option) (*Client, *sleepRecorder) {
	t.Helper()
	all := append([]Option{WithAPIKey(testKey), WithBaseURL(fs.srv.URL + "/v1/")}, opts...)
	c, err := NewClient(all...)
	if err != nil {
		t.Fatal(err)
	}
	rec := &sleepRecorder{}
	c.sleep = rec.sleep
	return c, rec
}
