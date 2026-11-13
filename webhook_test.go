package ayinbox

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

type sigVector struct {
	Secret     string `json:"secret"`
	Timestamp  int64  `json:"timestamp"`
	Body       string `json:"body"`
	ExpectedV1 string `json:"expected_v1"`
	Header     string `json:"header"`
}

func loadVector(t *testing.T) sigVector {
	t.Helper()
	var v sigVector
	if err := json.Unmarshal(readFixture(t, "webhook_signature.json"), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestWebhookSignatureVector(t *testing.T) {
	v := loadVector(t)
	now := time.Unix(v.Timestamp, 0)
	if !strings.HasSuffix(v.Header, v.ExpectedV1) {
		t.Fatal("vector header and expected_v1 disagree")
	}
	if err := VerifyWebhookSignatureAt([]byte(v.Body), v.Header, v.Secret, 0, now); err != nil {
		t.Fatalf("valid vector refused: %v", err)
	}
	tampered := strings.Replace(v.Body, "reply_received", "reply_receivee", 1)
	if err := VerifyWebhookSignatureAt([]byte(tampered), v.Header, v.Secret, 0, now); !errors.Is(err, ErrInvalidSignature) || !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("tampered body: %v", err)
	}
	if err := VerifyWebhookSignatureAt([]byte(v.Body), v.Header, "whsec_wrong", 0, now); !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("wrong secret: %v", err)
	}
	if err := VerifyWebhookSignatureAt([]byte(v.Body), v.Header, strings.TrimPrefix(v.Secret, "whsec_"), 0, now); err == nil {
		t.Fatal("secret without its prefix must not verify")
	}
	if err := VerifyWebhookSignatureAt([]byte(v.Body), v.Header, v.Secret, 0, now.Add(301*time.Second)); !errors.Is(err, ErrSignatureExpired) {
		t.Fatalf("expired: %v", err)
	}
	if err := VerifyWebhookSignatureAt([]byte(v.Body), v.Header, v.Secret, 0, now.Add(300*time.Second)); err != nil {
		t.Fatalf("edge of tolerance: %v", err)
	}
	if err := VerifyWebhookSignatureAt([]byte(v.Body), v.Header, v.Secret, time.Hour, now.Add(30*time.Minute)); err != nil {
		t.Fatalf("custom tolerance: %v", err)
	}
	multi := "t=1700000000,v1=" + strings.Repeat("0", 64) + ",v0=abc,v1=" + v.ExpectedV1
	if err := VerifyWebhookSignatureAt([]byte(v.Body), multi, v.Secret, 0, now); err != nil {
		t.Fatalf("second v1 should match: %v", err)
	}
	for _, h := range []string{"", "t=1700000000", "v1=" + v.ExpectedV1, "t=abc,v1=" + v.ExpectedV1, "garbage"} {
		if err := VerifyWebhookSignatureAt([]byte(v.Body), h, v.Secret, 0, now); !errors.Is(err, ErrSignatureHeader) {
			t.Fatalf("header %q: %v", h, err)
		}
	}
	if err := VerifyWebhookSignature([]byte(v.Body), v.Header, v.Secret); !errors.Is(err, ErrSignatureExpired) {
		t.Fatalf("a 2023 timestamp must be outside the tolerance today: %v", err)
	}
}

func TestConstructWebhookEvent(t *testing.T) {
	v := loadVector(t)
	if _, err := ConstructWebhookEvent([]byte(v.Body), v.Header, v.Secret); err == nil {
		t.Fatal("old vector should fail on tolerance")
	}
	ts := time.Now().Unix()
	hdr := signForTest(v.Secret, ts, v.Body)
	ev, err := ConstructWebhookEvent([]byte(v.Body), hdr, v.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "campaign.reply_received" || ev.ID != "00000000-0000-4000-8000-000000000001" || string(ev.Data) != "{}" {
		t.Fatalf("event %+v", ev)
	}
	c, _ := NewClient(WithAPIKey(testKey))
	if err := c.Webhooks.VerifySignature([]byte(v.Body), hdr, v.Secret); err != nil {
		t.Fatal(err)
	}
}

func signForTest(secret string, ts int64, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + body))
	return "t=" + strconv.FormatInt(ts, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}
