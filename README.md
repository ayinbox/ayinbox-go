# AY Inbox Go SDK

The official Go client for [AY Inbox](https://ayinbox.com), a hosted cold email platform. It sends cold email at a paced daily volume, brings every reply into one inbox, drafts answers from a per-campaign brain and can offer a meeting time. This package calls its public REST API at `https://api.ayinbox.com/v1`.

AY Inbox is a hosted service. This repository holds the Go client library for its API. The service itself is not open source.

Access to the hosted service is invite only for now: ask at team@ayinbox.com.

- Standard library only, Go 1.22 or later
- `context.Context` first on every call
- Typed errors, automatic retries that honour `Retry-After`, idempotency keys, auto-paging and webhook signature verification

## Install

Not on pkg.go.dev yet: no version has been tagged. Until `v0.1.0` is released, install from the main branch:

```bash
go get github.com/ayinbox/ayinbox-go@main
```

```go
import "github.com/ayinbox/ayinbox-go"
```

## Quick start

Create an API key in the dashboard (`https://app.ayinbox.com`) and export it as `AYINBOX_API_KEY`.

```go
client, err := ayinbox.NewClient() // reads AYINBOX_API_KEY
if err != nil {
	log.Fatal(err)
}
ctx := context.Background()

me, err := client.Me.Get(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println("signed in as", me.Email, "in", me.OrganizationName)

pager := client.Campaigns.ListAutoPaging(nil)
for pager.Next(ctx) {
	fmt.Println(pager.Current().Name, pager.Current().Status)
}
if err := pager.Err(); err != nil {
	log.Fatal(err)
}

_, err = client.Contacts.Create(ctx,
	[]ayinbox.ContactInput{{Email: "lee@example.com", FirstName: "Lee"}},
	ayinbox.WithIdempotencyKey(ayinbox.NewIdempotencyKey()))
if errors.Is(err, ayinbox.ErrPermissionDenied) {
	log.Printf("this key lacks WRITE_CONTACTS: %v", err)
}
```

Runnable programs are in [`examples/`](examples/): `list-campaigns` and `verify-webhook`.

## Configuration

| Option | Default | Notes |
|---|---|---|
| `WithAPIKey(key)` | `AYINBOX_API_KEY` | An API key (`aybox_...`) or an OAuth access token (`ayat_...`). `NewClient` returns `ErrMissingAPIKey` when neither is set |
| `WithBaseURL(url)` | `AYINBOX_BASE_URL`, then `https://api.ayinbox.com/v1` | A trailing slash is removed |
| `WithTimeout(d)` | 30 seconds | Per attempt |
| `WithMaxRetries(n)` | 2 | 0 disables retries |
| `WithHTTPClient(hc)` | a new `*http.Client` | For proxies, custom transports or tracing |

Per call, pass any of `WithIdempotencyKey`, `WithRequestTimeout`, `WithHeader`, `WithRequestID` and `WithResponse` (which copies the raw response, including the `X-RateLimit-*` headers, into a `*ayinbox.Response`).

The key is sent only in the `Authorization` header. It never appears in an error, and printing the client shows it as `[redacted]`.

## Resources

| Service | Methods |
|---|---|
| `client.Me` | `Get` |
| `client.Campaigns` | `List`, `ListAutoPaging`, `Get`, `Create`, `Update`, `Delete`, `Start`, `Pause`, `Duplicate`, `Overview` |
| `client.Campaigns.Steps` | `List`, `Create`, `Update`, `Delete` |
| `client.Campaigns.Leads` | `List`, `ListAutoPaging`, `Add`, `GetHold`, `Pause`, `Resume` |
| `client.Contacts` | `List`, `ListAutoPaging`, `Get`, `Create`, `Update`, `Delete`, `BulkDelete` |
| `client.Mailboxes` | `List`, `ListAutoPaging`, `Get`, `GetSync` |
| `client.Unibox` | `ListThreads`, `ListThreadsAutoPaging`, `GetThread`, `GetThreadAutoPaging`, `GetMessage`, `UnreadCount`, `Reply`, `ListDrafts`, `SaveDraft`, `DeleteDraft`, `ListAgentDrafts`, `ApproveAgentDraft`, `DiscardAgentDraft` |
| `client.Templates` | `List`, `Get`, `Create`, `Update`, `Delete` |
| `client.Webhooks` | `List`, `Create`, `Update`, `Delete`, `RotateSecret`, `Verify`, `EventTypes`, `ListDeliveries`, `ListDeliveriesAutoPaging`, `VerifySignature` |
| `client.Analytics` | `Dashboard`, `Campaign`, `CampaignDaily`, `Accounts`, `AccountsAutoPaging`, `Account` |
| `client.Suppressions` | `List`, `ListAutoPaging`, `Add`, `Remove` |
| `client.AITools` | `List`, `Call` |

`client.Mailboxes` maps to the API's `/emails` routes. `Campaigns.Leads.List` and `Add` are wrappers over contact search and contact create with one campaign id.

Every route the table does not cover is reachable through `client.Request`, which goes through the same authentication, retries and error mapping:

```go
var plan map[string]any
_, err := client.Request(ctx, "GET", "/campaigns/"+id+"/send-plan", nil, nil, &plan)
```

`body` is encoded as JSON (a `[]byte` or `json.RawMessage` is sent as is), and the answer is decoded into `out` when it is not nil.

Response types name the fields the API documents. Every top-level object keeps its full JSON in a `Raw` field, so a field added to the API later is readable without an SDK update. Fields whose shape the API does not document are `map[string]any`, `[]any` or `json.RawMessage`.

## Errors

Every non-2xx answer and every network failure is an `*ayinbox.Error`:

```go
_, err := client.Campaigns.Get(ctx, id)
var apiErr *ayinbox.Error
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.Status, apiErr.Code, apiErr.Message, apiErr.RequestID)
}
switch {
case errors.Is(err, ayinbox.ErrNotFound):
case errors.Is(err, ayinbox.ErrRateLimited):
	fmt.Println("retry after", apiErr.RetryAfter)
}
```

| Sentinel | Status | Example codes |
|---|---|---|
| `ErrBadRequest`, `ErrUnprocessable` (both match `ErrValidation`) | 400, 422 | `bad_request`, `invalid_cursor`, `duplicate_query_parameter`, `unprocessable`, `idempotency_key_required` |
| `ErrAuthentication` | 401 | `unauthorized`, `api_key_holder_left`, `session_ended` |
| `ErrPaymentRequired` | 402 | `insufficient_credits` |
| `ErrPermissionDenied` | 403 | `forbidden`, `api_key_mailbox_limited`, `review_off_needs_send` |
| `ErrNotFound` | 404 | `not_found` |
| `ErrConflict` | 409 | `conflict`, `idempotency_outcome_unknown`, `idempotent_secret_withheld`, `contact_email_taken`, `draft_flagged` |
| `ErrRateLimited` | 429 | `rate_limit_exceeded`, `usage_cap_exceeded` |
| `ErrServiceUnavailable` | 503 | `service_unavailable` |
| `ErrInternalServer` | 500, 501, 502, 504, other 5xx | `internal_error`, `not_implemented` |
| `ErrServer` | any 5xx | |
| `ErrConnection`, `ErrTimeout` | none | network failure, DNS, TLS, attempt timeout |

There are matching predicates (`IsNotFound`, `IsRateLimited`, `IsBadRequest`, ...) and `ErrorCode(err)`. Branch on `Status` and `Code`, never on `Message`. A cancelled or expired `ctx` returns `ctx.Err()` unchanged.

Some 429 answers carry no `code` and no `request_id`. The SDK fills `Code` with `rate_limit_exceeded`, takes the request id from the `X-Request-Id` header when there is one, and reads the wait from `Retry-After`, `retry_after` or `retry_after_ms`.

## Retries and timeouts

The client retries automatically, up to `WithMaxRetries` times (default 2, so 3 attempts):

- on 429, 502, 503 and 504, and on connection errors and timeouts
- never on 500 or on any 409, including `idempotency_outcome_unknown`
- a `POST` or `PATCH` without an idempotency key is retried only on 429, which the API answers before running anything. `GET`, `PUT` and `DELETE` are always safe to retry, and so is `Contacts.List`, a `POST` that only reads

The wait is `Retry-After` when the API sends it (capped at 120 seconds), otherwise exponential backoff with full jitter (base 0.5 seconds, cap 8 seconds). `WithTimeout` bounds each attempt; bound the whole call, retries included, with the `ctx` deadline.

## Pagination

List methods return one `*ayinbox.Page[T]` with `Data` and `Pagination` (`NextCursor`, `HasMore`, an optional `Total`). Pass `Pagination.Next()` back as `Cursor` for the next page, or let a `Pager` do it:

```go
pager := client.Contacts.ListAutoPaging(&ayinbox.ContactListParams{
	Search: ayinbox.ContactSearch{Query: "example.com", SortBy: "email"},
})
for pager.Next(ctx) {
	c := pager.Current()
	_ = c
}
err := pager.Err()
```

`pager.Collect(ctx)` returns every row in one slice. On Go 1.23 or later, `pager.All(ctx)` is a range-over-func iterator: `for c, err := range pager.All(ctx)`. `ayinbox.Paginate[T](client, method, path, query, body)` walks any paginated route. A contact search resends the same body on every page, because a cursor is bound to the sort it was issued under.

Recommended page sizes are 1 to 100. Templates, webhook endpoints, campaign steps and drafts are not paginated and return everything in one call.

## Idempotency

Every mutating method accepts `ayinbox.WithIdempotencyKey(key)`, sent as the `Idempotency-Key` header. The SDK does not generate keys on its own: pass one (for example `ayinbox.NewIdempotencyKey()`, a random UUIDv4) on any mutating call you might retry, and the SDK reuses it on its own retries of that call.

- the API stores the completed answer for 24 hours per workspace and key, and replays it for the same request
- the same key with a different request answers `409 conflict`
- when the first attempt ended without a definite outcome, the key answers `409 idempotency_outcome_unknown` for 24 hours. Establish whether the change happened before trying again with a new key
- an answer that held a secret shown once (creating a webhook endpoint, rotating its secret) is never replayed: a retry answers `409 idempotent_secret_withheld`

## Webhooks

Each delivery carries `X-Ayinbox-Signature: t=<unix-seconds>,v1=<hex>`. Verify it against the raw body, before parsing it, with the endpoint secret including its `whsec_` prefix:

```go
body, _ := io.ReadAll(r.Body)
ev, err := ayinbox.ConstructWebhookEvent(body, r.Header.Get(ayinbox.SignatureHeader), secret)
if err != nil { // errors.Is(err, ayinbox.ErrInvalidSignature)
	http.Error(w, "invalid signature", http.StatusBadRequest)
	return
}
// ev.ID equals X-Ayinbox-Event-Id and is stable across retries: use it to deduplicate.
```

`VerifyWebhookSignature` checks the signature only. Signatures are compared in constant time, and a timestamp more than 300 seconds from now is refused (`VerifyWebhookSignatureAt` takes another tolerance and clock).

## Calls that send email

These calls put real email on the wire:

- `Campaigns.Start` (`POST /campaigns/{id}/start`): needs `START_CAMPAIGNS` or `SEND_CAMPAIGNS`.
- `Campaigns.Update` with `Status` set to `active` (and `completed` or `paused_undeliverable`): needs `START_CAMPAIGNS` or `SEND_CAMPAIGNS` in addition to `WRITE_CAMPAIGNS`.
- `Campaigns.Leads.Resume` on a finished campaign: needs `START_CAMPAIGNS` or `SEND_CAMPAIGNS`.
- `Unibox.Reply` and `Unibox.ApproveAgentDraft`: need `WRITE_UNIBOX`. A key with `WRITE_UNIBOX` can send replies from any mailbox it can reach.
- Through `client.Request`: `POST /unibox/compose`, `POST /emails/{id}/send`, `POST /campaigns/{id}/test-email`, the first-email review queue (`/campaigns/{id}/reviews/...`) and placement tests need `SEND_CAMPAIGNS` (compose needs `WRITE_UNIBOX`).

Adding contacts can send mail without calling start: contacts added to an `active` campaign are queued, a `completed` campaign starts again on its own when leads are added, and a contact that comes to match a segment linked to a sending campaign is enrolled. This applies to `Contacts.Create` and `Campaigns.Leads.Add`.

`START_CAMPAIGNS` starts, resumes and pauses campaigns and nothing else. `SEND_CAMPAIGNS` covers everything `START_CAMPAIGNS` covers plus the other sending actions. `WRITE_CAMPAIGNS` alone edits drafts and pauses, and cannot start anything.

A key for a model that reads the inbox should use the "Agent: read and draft" preset (84574239): every read scope plus `AI_DRAFT`, with nothing that writes or sends.

Some calls spend workspace credits (AI drafts, research, enrichment, paid tools). They answer `402 insufficient_credits` when the balance is out.

Retrying a send without an idempotency key can send twice. Pass an idempotency key on every mutating call you might retry.

## Documentation

- API reference: https://docs.ayinbox.com/api/
- Product guides: https://docs.ayinbox.com/
- Package documentation: `go doc github.com/ayinbox/ayinbox-go`

## Security

Report a vulnerability to team@ayinbox.com, not in a public issue. See [SECURITY.md](SECURITY.md).

## License

MIT. See [LICENSE](LICENSE).
