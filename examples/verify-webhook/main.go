// Command verify-webhook is an HTTP server that checks the signature of every
// AY Inbox webhook delivery before it trusts the body.
//
//	AYINBOX_WEBHOOK_SECRET=whsec_... go run ./examples/verify-webhook
package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/ayinbox/ayinbox-go"
)

func main() {
	secret := os.Getenv("AYINBOX_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("set AYINBOX_WEBHOOK_SECRET to the endpoint's whsec_ secret")
	}
	http.HandleFunc("/ayinbox", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		ev, err := ayinbox.ConstructWebhookEvent(body, r.Header.Get(ayinbox.SignatureHeader), secret)
		if err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
		// Deduplicate on ev.ID: it is stable across retries of one event.
		log.Printf("event %s %s", ev.EventType, ev.ID)
		w.WriteHeader(http.StatusNoContent)
	})
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
