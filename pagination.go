package ayinbox

import (
	"context"
	"net/url"
)

// Pagination is the cursor block of a paginated list.
type Pagination struct {
	// NextCursor is the opaque cursor of the next page, or nil on the last page.
	// Pass it back unchanged; never parse it.
	NextCursor *string `json:"next_cursor"`
	// HasMore is true when another page exists.
	HasMore bool `json:"has_more"`
	// Total is the number of matching rows on the lists that report it.
	// Treat it as optional.
	Total *int64 `json:"total,omitempty"`
}

// Next returns the next cursor, or "" when there is no next page.
func (p Pagination) Next() string {
	if !p.HasMore || p.NextCursor == nil {
		return ""
	}
	return *p.NextCursor
}

// Page is one page of a paginated list.
type Page[T any] struct {
	// Data holds the rows of this page.
	Data []T `json:"data"`
	// Pagination holds the cursor of the next page.
	Pagination Pagination `json:"pagination"`
}

// Pager walks every page of a list, one row at a time.
//
//	pager := client.Campaigns.ListAutoPaging(nil)
//	for pager.Next(ctx) {
//		c := pager.Current()
//		// ...
//	}
//	if err := pager.Err(); err != nil { ... }
//
// A 429 during the walk is waited out by the client's retries.
type Pager[T any] struct {
	fetch  func(ctx context.Context, cursor string) ([]T, Pagination, error)
	buf    []T
	cur    T
	cursor string
	done   bool
	err    error
}

func newPager[T any](fetch func(ctx context.Context, cursor string) ([]T, Pagination, error)) *Pager[T] {
	return &Pager[T]{fetch: fetch}
}

// Next advances to the next row, fetching the next page when needed. It
// returns false at the end of the list or on an error; check Err afterwards.
func (p *Pager[T]) Next(ctx context.Context) bool {
	for len(p.buf) == 0 {
		if p.err != nil || p.done {
			return false
		}
		data, pg, err := p.fetch(ctx, p.cursor)
		if err != nil {
			p.err = err
			return false
		}
		p.buf = data
		if next := pg.Next(); next == "" || next == p.cursor {
			p.done = true
		} else {
			p.cursor = next
		}
	}
	p.cur = p.buf[0]
	p.buf = p.buf[1:]
	return true
}

// Current returns the row Next moved to.
func (p *Pager[T]) Current() T { return p.cur }

// Err returns the error that stopped the walk, if any.
func (p *Pager[T]) Err() error { return p.err }

// Collect walks the remaining rows and returns them all.
func (p *Pager[T]) Collect(ctx context.Context) ([]T, error) {
	var out []T
	for p.Next(ctx) {
		out = append(out, p.Current())
	}
	return out, p.Err()
}

// Paginate walks any endpoint that answers the data + pagination envelope.
// method is usually GET; for a POST search, body is resent unchanged on every
// page. Rows are decoded into T (use json.RawMessage or map[string]any for an
// untyped walk).
func Paginate[T any](c *Client, method, path string, query url.Values, body any, opts ...RequestOption) *Pager[T] {
	return newPager(func(ctx context.Context, cursor string) ([]T, Pagination, error) {
		q := url.Values{}
		for k, v := range query {
			q[k] = append([]string(nil), v...)
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		} else {
			q.Del("cursor")
		}
		var page Page[T]
		o := opts
		if method != "GET" {
			o = append(append([]RequestOption(nil), opts...), withRetrySafe())
		}
		if _, err := c.Request(ctx, method, path, q, body, &page, o...); err != nil {
			return nil, Pagination{}, err
		}
		return page.Data, page.Pagination, nil
	})
}
