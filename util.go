package ayinbox

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// String returns a pointer to v, for optional request fields.
func String(v string) *string { return &v }

// Bool returns a pointer to v, for optional request fields.
func Bool(v bool) *bool { return &v }

// Int returns a pointer to v, for optional request fields.
func Int(v int) *int { return &v }

// Float64 returns a pointer to v, for optional request fields.
func Float64(v float64) *float64 { return &v }

// Time returns a pointer to v, for optional request fields.
func Time(v time.Time) *time.Time { return &v }

// NullTime is a request timestamp that can be sent as an explicit JSON null.
// A nil *NullTime omits the field; SetTime sends a value; ClearTime sends null.
type NullTime struct {
	// Time is the value sent when Valid is true.
	Time time.Time
	// Valid is false to send JSON null.
	Valid bool
}

// SetTime returns a NullTime that sends t.
func SetTime(t time.Time) *NullTime { return &NullTime{Time: t, Valid: true} }

// ClearTime returns a NullTime that sends JSON null, clearing the field.
func ClearTime() *NullTime { return &NullTime{} }

// MarshalJSON encodes the time as RFC 3339, or null when not Valid.
func (n NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Time.UTC().Format(time.RFC3339Nano))
}

// UnmarshalJSON decodes an RFC 3339 string or null.
func (n *NullTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*n = NullTime{}
		return nil
	}
	if err := json.Unmarshal(b, &n.Time); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// encodeQuery turns a struct with `url:"name"` tags into query values.
// Zero values are skipped; slices are comma-joined so no parameter repeats.
func encodeQuery(params any) url.Values {
	q := url.Values{}
	if params == nil {
		return q
	}
	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return q
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return q
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Tag.Get("url")
		if name == "" || name == "-" {
			continue
		}
		f := v.Field(i)
		switch x := f.Interface().(type) {
		case string:
			if x != "" {
				q.Set(name, x)
			}
		case int:
			if x != 0 {
				q.Set(name, strconv.Itoa(x))
			}
		case bool:
			if x {
				q.Set(name, "true")
			}
		case *bool:
			if x != nil {
				q.Set(name, strconv.FormatBool(*x))
			}
		case *int:
			if x != nil {
				q.Set(name, strconv.Itoa(*x))
			}
		case []string:
			if len(x) > 0 {
				q.Set(name, strings.Join(x, ","))
			}
		case time.Time:
			if !x.IsZero() {
				q.Set(name, x.UTC().Format(time.RFC3339))
			}
		}
	}
	return q
}

// unmarshalKeepRaw decodes b into v and keeps a copy of b in raw.
func unmarshalKeepRaw(b []byte, v any, raw *json.RawMessage) error {
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	*raw = append(json.RawMessage(nil), b...)
	return nil
}
