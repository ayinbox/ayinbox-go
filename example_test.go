package ayinbox_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/ayinbox/ayinbox-go"
)

// fakeAPI stands in for https://api.ayinbox.com/v1 so the examples run offline.
func fakeAPI() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"user_id":"u1","email":"dana@example.com","name":"Dana Example","auth_type":"api_key","scopes":["read_campaigns"]}`)
	})
	mux.HandleFunc("/v1/campaigns", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"data":[{"id":"c1","name":"Q4 outreach","status":"active"}],"pagination":{"next_cursor":"c1_next","has_more":true}}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"c2","name":"Webinar follow-up","status":"draft"}],"pagination":{"next_cursor":null,"has_more":false}}`)
	})
	mux.HandleFunc("/v1/campaigns/missing", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"Not Found","message":"Resource not found.","code":"not_found","request_id":"req-1"}`)
	})
	return httptest.NewServer(mux)
}

func Example() {
	srv := fakeAPI()
	defer srv.Close()

	// In real code: ayinbox.NewClient() reads AYINBOX_API_KEY.
	client, err := ayinbox.NewClient(ayinbox.WithAPIKey("aybox_example"), ayinbox.WithBaseURL(srv.URL+"/v1"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	me, err := client.Me.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("signed in as", me.Email)

	pager := client.Campaigns.ListAutoPaging(nil)
	for pager.Next(ctx) {
		c := pager.Current()
		fmt.Println(c.Name, c.Status)
	}
	if err := pager.Err(); err != nil {
		log.Fatal(err)
	}

	_, err = client.Campaigns.Get(ctx, "missing")
	if errors.Is(err, ayinbox.ErrNotFound) {
		var apiErr *ayinbox.Error
		errors.As(err, &apiErr)
		fmt.Println("not found:", apiErr.Code, apiErr.RequestID)
	}
	// Output:
	// signed in as dana@example.com
	// Q4 outreach active
	// Webinar follow-up draft
	// not found: not_found req-1
}

func ExampleVerifyWebhookSignatureAt() {
	body := []byte(`{"id":"00000000-0000-4000-8000-000000000001","event_type":"campaign.reply_received","organization_id":"00000000-0000-4000-8000-0000000000aa","created_at":"2023-11-14T22:13:20Z","data":{}}`)
	header := "t=1700000000,v1=d1efe3010ae2523566985158ab4852f8f3569365a06941ef87e3a641e3171342"

	// In a handler, use VerifyWebhookSignature, which checks against the current time.
	err := ayinbox.VerifyWebhookSignatureAt(body, header, "whsec_test_secret", 0, time.Unix(1700000000, 0))
	fmt.Println("valid:", err == nil)

	err = ayinbox.VerifyWebhookSignatureAt(body, header, "whsec_other", 0, time.Unix(1700000000, 0))
	fmt.Println("wrong secret:", errors.Is(err, ayinbox.ErrInvalidSignature))
	// Output:
	// valid: true
	// wrong secret: true
}

func ExampleWithIdempotencyKey() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"id":"ct1","email":"lee@example.com","created_at":"2026-11-01T09:00:00Z","updated_at":"2026-11-01T09:00:00Z"}]`)
		fmt.Println("Idempotency-Key sent:", r.Header.Get("Idempotency-Key") != "")
	}))
	defer srv.Close()
	client, _ := ayinbox.NewClient(ayinbox.WithAPIKey("aybox_example"), ayinbox.WithBaseURL(srv.URL+"/v1"))

	contacts, err := client.Contacts.Create(context.Background(),
		[]ayinbox.ContactInput{{Email: "lee@example.com", FirstName: "Lee"}},
		ayinbox.WithIdempotencyKey(ayinbox.NewIdempotencyKey()))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("created", contacts[0].Email)
	// Output:
	// Idempotency-Key sent: true
	// created lee@example.com
}
