package ayinbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Sentinel errors for errors.Is. An *Error matches the sentinel of its category.
var (
	// ErrBadRequest matches a 400 answer.
	ErrBadRequest = errors.New("ayinbox: bad request")
	// ErrUnprocessable matches a 422 answer.
	ErrUnprocessable = errors.New("ayinbox: unprocessable")
	// ErrValidation matches both 400 and 422 answers.
	ErrValidation = errors.New("ayinbox: validation failed")
	// ErrAuthentication matches a 401 answer.
	ErrAuthentication = errors.New("ayinbox: authentication failed")
	// ErrPaymentRequired matches a 402 answer (insufficient credits).
	ErrPaymentRequired = errors.New("ayinbox: payment required")
	// ErrPermissionDenied matches a 403 answer.
	ErrPermissionDenied = errors.New("ayinbox: permission denied")
	// ErrNotFound matches a 404 answer.
	ErrNotFound = errors.New("ayinbox: not found")
	// ErrConflict matches a 409 answer.
	ErrConflict = errors.New("ayinbox: conflict")
	// ErrRateLimited matches a 429 answer.
	ErrRateLimited = errors.New("ayinbox: rate limited")
	// ErrServiceUnavailable matches a 503 answer.
	ErrServiceUnavailable = errors.New("ayinbox: service unavailable")
	// ErrInternalServer matches any 5xx answer other than 503.
	ErrInternalServer = errors.New("ayinbox: internal server error")
	// ErrServer matches any 5xx answer, 503 included.
	ErrServer = errors.New("ayinbox: server error")
	// ErrConnection matches a network failure where no HTTP answer arrived.
	ErrConnection = errors.New("ayinbox: connection error")
	// ErrTimeout matches an attempt that ran out of time. It also matches ErrConnection.
	ErrTimeout = errors.New("ayinbox: request timed out")
)

// Error is returned for every non-2xx answer and every network failure.
// It never holds the API key.
type Error struct {
	// Status is the HTTP status, or 0 for a network failure.
	Status int
	// Code is the API's stable machine key, such as "not_found" or
	// "api_key_holder_left". Branch on Code and Status, never on Message.
	Code string
	// Reason is a finer key the API sets on a few codes. It may be empty.
	Reason string
	// Message is the human text from the API.
	Message string
	// RequestID identifies the request in AY Inbox support and logs.
	RequestID string
	// RetryAfter is how long the API asked the caller to wait, when it said so.
	RetryAfter time.Duration
	// Body is the raw response body.
	Body []byte
	// Header holds the response headers.
	Header http.Header

	timeout bool
	err     error
}

// Error formats the status, code, message and request id.
func (e *Error) Error() string {
	if e.Status == 0 {
		if e.timeout {
			return "ayinbox: request timed out: " + e.Message
		}
		return "ayinbox: connection error: " + e.Message
	}
	s := fmt.Sprintf("ayinbox: %d %s", e.Status, e.Code)
	if e.Message != "" {
		s += ": " + e.Message
	}
	if e.RequestID != "" {
		s += " (request_id " + e.RequestID + ")"
	}
	return s
}

// Unwrap returns the underlying network error, if any.
func (e *Error) Unwrap() error { return e.err }

// Is reports whether e belongs to the category of the sentinel target.
func (e *Error) Is(target error) bool {
	switch target {
	case ErrConnection:
		return e.Status == 0
	case ErrTimeout:
		return e.Status == 0 && e.timeout
	case ErrBadRequest:
		return e.Status == 400
	case ErrUnprocessable:
		return e.Status == 422
	case ErrValidation:
		return e.Status == 400 || e.Status == 422
	case ErrAuthentication:
		return e.Status == 401
	case ErrPaymentRequired:
		return e.Status == 402
	case ErrPermissionDenied:
		return e.Status == 403
	case ErrNotFound:
		return e.Status == 404
	case ErrConflict:
		return e.Status == 409
	case ErrRateLimited:
		return e.Status == 429
	case ErrServiceUnavailable:
		return e.Status == 503
	case ErrInternalServer:
		return e.Status >= 500 && e.Status != 503
	case ErrServer:
		return e.Status >= 500
	}
	return false
}

// IsBadRequest reports whether err is a 400 or 422 validation error.
func IsBadRequest(err error) bool { return errors.Is(err, ErrValidation) }

// IsAuthentication reports whether err is a 401.
func IsAuthentication(err error) bool { return errors.Is(err, ErrAuthentication) }

// IsPaymentRequired reports whether err is a 402.
func IsPaymentRequired(err error) bool { return errors.Is(err, ErrPaymentRequired) }

// IsPermissionDenied reports whether err is a 403.
func IsPermissionDenied(err error) bool { return errors.Is(err, ErrPermissionDenied) }

// IsNotFound reports whether err is a 404.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// IsConflict reports whether err is a 409.
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }

// IsRateLimited reports whether err is a 429.
func IsRateLimited(err error) bool { return errors.Is(err, ErrRateLimited) }

// IsServerError reports whether err is any 5xx.
func IsServerError(err error) bool { return errors.Is(err, ErrServer) }

// IsConnection reports whether err is a network failure or timeout.
func IsConnection(err error) bool { return errors.Is(err, ErrConnection) }

// ErrorCode returns the API code of err, or "" when err is not an *Error.
func ErrorCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

var codePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

var defaultCodes = map[int]string{
	400: "bad_request",
	401: "unauthorized",
	402: "insufficient_credits",
	403: "forbidden",
	404: "not_found",
	409: "conflict",
	422: "unprocessable",
	429: "rate_limit_exceeded",
	500: "internal_error",
	501: "not_implemented",
	503: "service_unavailable",
}

func defaultCode(status int) string {
	if c, ok := defaultCodes[status]; ok {
		return c
	}
	if status >= 500 {
		return "internal_error"
	}
	return http.StatusText(status)
}

type errorEnvelope struct {
	Error        string   `json:"error"`
	Message      string   `json:"message"`
	Code         string   `json:"code"`
	Reason       string   `json:"reason"`
	RequestID    string   `json:"request_id"`
	RetryAfter   *float64 `json:"retry_after"`
	RetryAfterMs *float64 `json:"retry_after_ms"`
}

func parseError(r *Response) *Error {
	body := r.Body
	if len(body) > maxErrorBodySize {
		body = body[:maxErrorBodySize]
	}
	e := &Error{Status: r.StatusCode, Body: body, Header: r.Header}
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil {
		e.Message = env.Message
		e.Reason = env.Reason
		e.RequestID = env.RequestID
		switch {
		case env.Code != "":
			e.Code = env.Code
		case codePattern.MatchString(env.Error):
			e.Code = env.Error
		}
		if e.Message == "" {
			e.Message = env.Error
		}
		switch {
		case env.RetryAfter != nil && *env.RetryAfter > 0:
			e.RetryAfter = time.Duration(*env.RetryAfter * float64(time.Second))
		case env.RetryAfterMs != nil && *env.RetryAfterMs > 0:
			e.RetryAfter = time.Duration(math.Ceil(*env.RetryAfterMs/1000)) * time.Second
		}
	} else {
		e.Message = strings.TrimSpace(string(body))
	}
	if e.Code == "" {
		e.Code = defaultCode(r.StatusCode)
	}
	if e.RequestID == "" {
		e.RequestID = r.Header.Get(requestIDHdr)
	}
	if ra := parseRetryAfter(r.Header.Get("Retry-After")); ra > 0 {
		e.RetryAfter = ra
	}
	return e
}

func networkError(err error, parent, attempt context.Context) *Error {
	timeout := false
	if parent.Err() == nil && attempt.Err() == context.DeadlineExceeded {
		timeout = true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		timeout = true
	}
	return &Error{Message: err.Error(), timeout: timeout, err: err}
}
