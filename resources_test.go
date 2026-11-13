package ayinbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	campID    = "11111111-1111-4111-8111-111111111111"
	contactID = "22222222-2222-4222-8222-111111111111"
	mailboxID = "33333333-3333-4333-8333-111111111111"
	stepID    = "44444444-4444-4444-8444-111111111111"
	msgID     = "55555555-5555-4555-8555-111111111111"
	hookID    = "77777777-7777-4777-8777-111111111111"
	draftID   = "bbbbbbbb-bbbb-4bbb-8bbb-111111111111"
)

// expectCall checks the method, path and (when want is not empty) a JSON body of the last request.
func expectCall(t *testing.T, fs *fixtureServer, method, path, want string) recorded {
	t.Helper()
	r := fs.last()
	if r.Method != method || r.Path != "/v1"+path {
		t.Fatalf("got %s %s, want %s /v1%s", r.Method, r.Path, method, path)
	}
	if want != "" {
		var got, exp any
		if err := json.Unmarshal(r.Body, &got); err != nil {
			t.Fatalf("body %q: %v", r.Body, err)
		}
		if err := json.Unmarshal([]byte(want), &exp); err != nil {
			t.Fatal(err)
		}
		g, _ := json.Marshal(got)
		e, _ := json.Marshal(exp)
		if string(g) != string(e) {
			t.Fatalf("body %s, want %s", g, e)
		}
	}
	return r
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCampaigns(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	page, err := c.Campaigns.List(ctx, &CampaignListParams{Q: "q4", Status: "active", Limit: 2})
	must(t, err)
	if len(page.Data) != 2 || page.Pagination.Next() == "" || page.Pagination.Total == nil || *page.Pagination.Total != 3 {
		t.Fatalf("page %+v", page.Pagination)
	}
	if q := expectCall(t, fs, "GET", "/campaigns", "").Query; q != "limit=2&q=q4&status=active" {
		t.Fatalf("query %q", q)
	}

	camp, err := c.Campaigns.Get(ctx, campID)
	must(t, err)
	expectCall(t, fs, "GET", "/campaigns/"+campID, "")
	if camp.Name != "Q4 outreach" || camp.DailyLimit != 50 || len(camp.ScheduleWindows) != 7 || len(camp.Senders) != 1 || camp.StartDate != nil {
		t.Fatalf("campaign %+v", camp)
	}
	if camp.GuardrailBounceRateMax != 0.05 || camp.CreatedAt.IsZero() || len(camp.Raw) == 0 {
		t.Fatal("campaign fields")
	}

	_, err = c.Campaigns.Create(ctx, &CampaignCreateParams{
		Name: "New", DailyLimit: Int(30), SenderStrategy: String("explicit"),
		Senders: []CampaignSenderInput{{EmailAccountID: mailboxID}},
		Steps:   []StepInput{{Subject: "Hi", BodyPlain: "Hello", WaitAfter: Int(0)}},
	})
	must(t, err)
	expectCall(t, fs, "POST", "/campaigns", `{"name":"New","daily_limit":30,"sender_strategy":"explicit","senders":[{"email_account_id":"`+mailboxID+`"}],"steps":[{"subject":"Hi","body_plain":"Hello","wait_after":0}]}`)

	start := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	_, err = c.Campaigns.Update(ctx, campID, &CampaignUpdateParams{Name: String("Renamed"), StartDate: SetTime(start), EndDate: ClearTime()})
	must(t, err)
	expectCall(t, fs, "PATCH", "/campaigns/"+campID, `{"name":"Renamed","start_date":"2026-12-01T09:00:00Z","end_date":null}`)

	must(t, c.Campaigns.Delete(ctx, campID))
	expectCall(t, fs, "DELETE", "/campaigns/"+campID, "")

	st, err := c.Campaigns.Start(ctx, campID, &CampaignStartParams{AcknowledgeListRisk: true}, WithIdempotencyKey("s1"))
	must(t, err)
	if st.Status != "started" || st.WaitingForLeads {
		t.Fatalf("start %+v", st)
	}
	expectCall(t, fs, "POST", "/campaigns/"+campID+"/start", `{"acknowledge_list_risk":true}`)

	if _, err := c.Campaigns.Start(ctx, campID, nil); err != nil {
		t.Fatal(err)
	}
	if b := fs.last().Body; len(b) != 0 {
		t.Fatalf("start without params sent %q", b)
	}

	ps, err := c.Campaigns.Pause(ctx, campID)
	must(t, err)
	if ps.Status != "stopped" {
		t.Fatal(ps.Status)
	}
	expectCall(t, fs, "POST", "/campaigns/"+campID+"/stop", "")

	dup, err := c.Campaigns.Duplicate(ctx, campID, &CampaignDuplicateParams{Name: "Copy"})
	must(t, err)
	if dup.ID != campID {
		t.Fatal("duplicate")
	}
	expectCall(t, fs, "POST", "/campaigns/"+campID+"/duplicate", `{"name":"Copy"}`)

	ov, err := c.Campaigns.Overview(ctx)
	must(t, err)
	if ov.Total != 3 || ov.Active != 1 {
		t.Fatalf("overview %+v", ov)
	}
	expectCall(t, fs, "GET", "/campaigns-overview", "")
}

func TestCampaignSteps(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	steps, err := c.Campaigns.Steps.List(ctx, campID)
	must(t, err)
	if len(steps) != 2 || steps[1].WaitAfter != 3 || steps[1].Kind != "email" || !steps[1].ThreadReply {
		t.Fatalf("steps %+v", steps)
	}
	expectCall(t, fs, "GET", "/campaigns/"+campID+"/steps", "")

	_, err = c.Campaigns.Steps.Create(ctx, campID, nil)
	must(t, err)
	expectCall(t, fs, "POST", "/campaigns/"+campID+"/steps", `{}`)

	s, err := c.Campaigns.Steps.Update(ctx, campID, stepID, &StepParams{Subject: String("New subject"), Conditions: map[string]any{}})
	must(t, err)
	if s.ID == "" {
		t.Fatal("step id")
	}
	expectCall(t, fs, "PATCH", "/campaigns/"+campID+"/steps/"+stepID, `{"subject":"New subject","conditions":{}}`)

	must(t, c.Campaigns.Steps.Delete(ctx, campID, stepID))
	expectCall(t, fs, "DELETE", "/campaigns/"+campID+"/steps/"+stepID, "")
}

func TestCampaignLeads(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	leads, err := c.Campaigns.Leads.List(ctx, campID, &LeadListParams{LeadStatus: "active", Limit: 10})
	must(t, err)
	if len(leads.Data) != 1 || leads.Data[0].CampaignLead == nil || leads.Data[0].CampaignLead.Sent != 1 {
		t.Fatalf("leads %+v", leads.Data)
	}
	r := expectCall(t, fs, "POST", "/contacts/search", `{"campaign_ids":["`+campID+`"],"lead_status":"active"}`)
	if r.Query != "limit=10" {
		t.Fatalf("query %q", r.Query)
	}

	all, err := c.Campaigns.Leads.ListAutoPaging(campID, nil).Collect(ctx)
	must(t, err)
	if len(all) != 1 {
		t.Fatal("lead walk")
	}

	in := []ContactInput{{Email: "lee@example.com", Campaigns: []string{"other"}}}
	added, err := c.Campaigns.Leads.Add(ctx, campID, in, WithIdempotencyKey("lead-1"))
	must(t, err)
	if len(added) != 1 || len(in[0].Campaigns) != 1 {
		t.Fatal("add changed the caller's slice or returned nothing")
	}
	expectCall(t, fs, "POST", "/contacts", `[{"email":"lee@example.com","campaigns":["other","`+campID+`"]}]`)

	hold, err := c.Campaigns.Leads.GetHold(ctx, campID, contactID)
	must(t, err)
	if hold.Hold == nil || hold.Hold.Source != "manual" || hold.Hold.Until == nil {
		t.Fatalf("hold %+v", hold)
	}
	expectCall(t, fs, "GET", "/campaigns/"+campID+"/leads/"+contactID+"/hold", "")

	until := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	_, err = c.Campaigns.Leads.Pause(ctx, campID, contactID, &LeadPauseParams{Until: &until, Reason: "Holiday"})
	must(t, err)
	expectCall(t, fs, "POST", "/campaigns/"+campID+"/leads/"+contactID+"/pause", `{"until":"2026-11-20T00:00:00Z","reason":"Holiday"}`)

	_, err = c.Campaigns.Leads.Resume(ctx, campID, contactID)
	must(t, err)
	expectCall(t, fs, "POST", "/campaigns/"+campID+"/leads/"+contactID+"/resume", "")
}

func TestContacts(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	list, err := c.Contacts.List(ctx, &ContactListParams{Category: "cat", Search: ContactSearch{Query: "lee", Subscribed: Bool(true)}})
	must(t, err)
	if len(list.Data) == 0 {
		t.Fatal("no contacts")
	}
	r := expectCall(t, fs, "POST", "/contacts/search", `{"query":"lee","subscribed":true}`)
	if r.Query != "category=cat" || r.Header.Get("Idempotency-Key") != "" {
		t.Fatalf("query %q", r.Query)
	}
	_, err = c.Contacts.List(ctx, nil)
	must(t, err)
	expectCall(t, fs, "POST", "/contacts/search", `{}`)

	d, err := c.Contacts.Get(ctx, contactID)
	must(t, err)
	if d.Email != "lee@example.com" || d.Source != "api" || d.SourceDetail != "Example key" || d.FirstSeenAt == nil || d.CustomFields["title"] != "Head of Ops" || len(d.Raw) == 0 {
		t.Fatalf("detail %+v", d)
	}
	expectCall(t, fs, "GET", "/contacts/"+contactID, "")

	created, err := c.Contacts.Create(ctx, []ContactInput{{Email: "lee@example.com", FirstName: "Lee", CustomFields: map[string]string{"title": "Head of Ops"}}})
	must(t, err)
	if len(created) != 1 || created[0].Categories[0].Title != "VIP" {
		t.Fatalf("created %+v", created)
	}
	expectCall(t, fs, "POST", "/contacts", `[{"email":"lee@example.com","first_name":"Lee","custom_fields":{"title":"Head of Ops"}}]`)

	up, err := c.Contacts.Update(ctx, contactID, &ContactUpdateParams{Company: String("New Co"), AddCategories: []string{"x"}})
	must(t, err)
	if up.ID != contactID {
		t.Fatal("update")
	}
	expectCall(t, fs, "PATCH", "/contacts/"+contactID, `{"company":"New Co","add_categories":["x"]}`)

	must(t, c.Contacts.Delete(ctx, contactID))
	expectCall(t, fs, "DELETE", "/contacts/"+contactID, "")

	must(t, c.Contacts.BulkDelete(ctx, ContactBulkDelete{IDs: []string{"a", "b"}}))
	expectCall(t, fs, "DELETE", "/contacts", `["a","b"]`)
	must(t, c.Contacts.BulkDelete(ctx, ContactBulkDelete{All: true, Filters: &ContactSearch{Query: "old"}, Exclude: []string{"c"}}))
	expectCall(t, fs, "DELETE", "/contacts", `{"all":true,"filters":{"query":"old"},"exclude":["c"]}`)
}

func TestMailboxes(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	page, err := c.Mailboxes.List(ctx, &MailboxListParams{Tag: "t"})
	must(t, err)
	if len(page.Data) == 0 {
		t.Fatal("no mailboxes")
	}
	expectCall(t, fs, "GET", "/emails", "")

	mb, err := c.Mailboxes.Get(ctx, mailboxID)
	must(t, err)
	if mb.Provider != "gmail" || mb.WorkerID != nil || mb.Warmup == nil || mb.MinWaitTime != 600 || !strings.Contains(string(mb.Raw), "warmup_days") {
		t.Fatalf("mailbox %+v", mb)
	}
	expectCall(t, fs, "GET", "/emails/"+mailboxID, "")

	sync, err := c.Mailboxes.GetSync(ctx, mailboxID)
	must(t, err)
	if sync.State == nil || sync.State.BackfillSynced != 1200 || sync.Policy.DailyMessages != 2000 {
		t.Fatalf("sync %+v", sync)
	}
	expectCall(t, fs, "GET", "/emails/"+mailboxID+"/sync", "")

	all, err := c.Mailboxes.ListAutoPaging(nil).Collect(ctx)
	must(t, err)
	if len(all) == 0 {
		t.Fatal("walk")
	}
}

func TestUnibox(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	threads, err := c.Unibox.ListThreads(ctx, &ThreadListParams{Unseen: true, Automated: Bool(false), EmailIDs: []string{"a", "b"}, Folder: "inbox"})
	must(t, err)
	if len(threads.Data) != 1 || threads.Data[0].MessageCount != 2 || threads.Pagination.Total != nil {
		t.Fatalf("threads %+v", threads)
	}
	if q := expectCall(t, fs, "GET", "/unibox", "").Query; q != "automated=false&email_ids=a%2Cb&folder=inbox&unseen=true" {
		t.Fatalf("query %q", q)
	}

	th, err := c.Unibox.GetThread(ctx, &ThreadParams{ThreadID: "thread-example-001", EmailID: mailboxID})
	must(t, err)
	if len(th.Data) == 0 {
		t.Fatal("thread")
	}
	if q := expectCall(t, fs, "GET", "/unibox/thread", "").Query; !strings.Contains(q, "thread_id=thread-example-001") {
		t.Fatalf("query %q", q)
	}
	_ = c.Unibox.GetThreadAutoPaging(&ThreadParams{ThreadID: "x"})
	_ = c.Unibox.ListThreadsAutoPaging(nil)

	m, err := c.Unibox.GetMessage(ctx, msgID)
	must(t, err)
	if m.MessageID != "<msg-2@example.com>" || len(m.InReplyTo) != 1 || m.ReplyTo == nil || m.Size != 2048 {
		t.Fatalf("message %+v", m)
	}
	expectCall(t, fs, "GET", "/unibox/"+msgID, "")

	n, err := c.Unibox.UnreadCount(ctx, mailboxID)
	must(t, err)
	if r := expectCall(t, fs, "GET", "/unibox/count", ""); r.Query != "email_id="+mailboxID {
		t.Fatalf("query %q", r.Query)
	}
	_ = n

	sent, err := c.Unibox.Reply(ctx, &ReplyParams{EmailAccountID: mailboxID, To: []string{"lee@example.com"}, Subject: "Re: hi", BodyPlain: "Thanks"}, WithIdempotencyKey("r1"))
	must(t, err)
	if sent.TaskID == "" {
		t.Fatal("task id")
	}
	expectCall(t, fs, "POST", "/unibox/reply", `{"email_account_id":"`+mailboxID+`","to":["lee@example.com"],"subject":"Re: hi","body_plain":"Thanks"}`)

	drafts, err := c.Unibox.ListDrafts(ctx)
	must(t, err)
	if len(drafts) != 1 || drafts[0].Subject != "Following up" {
		t.Fatalf("drafts %+v", drafts)
	}
	expectCall(t, fs, "GET", "/unibox/drafts", "")

	id, err := c.Unibox.SaveDraft(ctx, draftID, &DraftParams{Subject: "Following up", Body: "Hi Lee,"})
	must(t, err)
	if id != draftID {
		t.Fatal(id)
	}
	expectCall(t, fs, "PUT", "/unibox/drafts/"+draftID, `{"subject":"Following up","body":"Hi Lee,"}`)

	must(t, c.Unibox.DeleteDraft(ctx, draftID))
	expectCall(t, fs, "DELETE", "/unibox/drafts/"+draftID, "")

	ad, err := c.Unibox.ListAgentDrafts(ctx)
	must(t, err)
	if len(ad) != 1 || ad[0].Confidence != 0.91 || ad[0].Grounding == nil {
		t.Fatalf("agent drafts %+v", ad)
	}
	expectCall(t, fs, "GET", "/unibox/agent-drafts", "")

	_, err = c.Unibox.ApproveAgentDraft(ctx, ad[0].ID, &ApproveDraftParams{AcknowledgeFlags: true}, WithIdempotencyKey("a1"))
	must(t, err)
	expectCall(t, fs, "POST", "/unibox/agent-drafts/"+ad[0].ID+"/approve", `{"acknowledge_flags":true}`)

	st, err := c.Unibox.DiscardAgentDraft(ctx, ad[0].ID)
	must(t, err)
	if st != "discarded" {
		t.Fatal(st)
	}
	expectCall(t, fs, "POST", "/unibox/agent-drafts/"+ad[0].ID+"/discard", "")
}

func TestTemplates(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	list, err := c.Templates.List(ctx, "follow")
	must(t, err)
	if len(list) == 0 {
		t.Fatal("no templates")
	}
	if r := expectCall(t, fs, "GET", "/templates", ""); r.Query != "q=follow" {
		t.Fatalf("query %q", r.Query)
	}
	tp, err := c.Templates.Get(ctx, "tpl")
	must(t, err)
	if tp.Name == "" {
		t.Fatal("template")
	}
	expectCall(t, fs, "GET", "/templates/tpl", "")
	_, err = c.Templates.Create(ctx, &TemplateCreateParams{Name: "T", Subject: "S"})
	must(t, err)
	expectCall(t, fs, "POST", "/templates", `{"name":"T","subject":"S"}`)
	_, err = c.Templates.Update(ctx, "tpl", &TemplateUpdateParams{BodyPlain: String("")})
	must(t, err)
	expectCall(t, fs, "PATCH", "/templates/tpl", `{"body_plain":""}`)
	must(t, c.Templates.Delete(ctx, "tpl"))
	expectCall(t, fs, "DELETE", "/templates/tpl", "")
}

func TestWebhooks(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	list, err := c.Webhooks.List(ctx)
	must(t, err)
	if len(list.Endpoints) == 0 || len(list.EventTypes) == 0 {
		t.Fatalf("list %+v", list)
	}
	expectCall(t, fs, "GET", "/webhooks", "")

	ep, err := c.Webhooks.Create(ctx, &WebhookParams{URL: "https://hooks.example.com/ayinbox", EventTypes: []string{"campaign.reply_received"}})
	must(t, err)
	if !strings.HasPrefix(ep.Secret, "whsec_") || ep.ID != hookID {
		t.Fatalf("endpoint %+v", ep)
	}
	expectCall(t, fs, "POST", "/webhooks", `{"url":"https://hooks.example.com/ayinbox","event_types":["campaign.reply_received"]}`)

	up, err := c.Webhooks.Update(ctx, hookID, &WebhookParams{URL: "https://hooks.example.com/v2", Enabled: Bool(false)})
	must(t, err)
	if up.Secret != "" {
		t.Fatal("update should carry no secret")
	}
	expectCall(t, fs, "PATCH", "/webhooks/"+hookID, `{"url":"https://hooks.example.com/v2","enabled":false}`)

	must(t, c.Webhooks.Delete(ctx, hookID))
	expectCall(t, fs, "DELETE", "/webhooks/"+hookID, "")

	sec, err := c.Webhooks.RotateSecret(ctx, hookID)
	must(t, err)
	if !strings.HasPrefix(sec, "whsec_") {
		t.Fatal(sec)
	}
	expectCall(t, fs, "POST", "/webhooks/"+hookID+"/rotate-secret", "")

	st, err := c.Webhooks.Verify(ctx, hookID)
	must(t, err)
	if st != "challenge_sent" {
		t.Fatal(st)
	}
	expectCall(t, fs, "POST", "/webhooks/"+hookID+"/verify", "")

	types, err := c.Webhooks.EventTypes(ctx)
	must(t, err)
	if len(types) == 0 {
		t.Fatal("event types")
	}
	expectCall(t, fs, "GET", "/webhooks/event-types", "")

	dl, err := c.Webhooks.ListDeliveries(ctx, &DeliveryListParams{Status: "delivered", EndpointID: hookID})
	must(t, err)
	if len(dl.Data) != 1 || dl.Data[0].ResponseStatus == nil || *dl.Data[0].ResponseStatus != 200 || dl.Pagination.Next() != "o1_NTA" {
		t.Fatalf("deliveries %+v", dl)
	}
	if q := expectCall(t, fs, "GET", "/webhooks/deliveries", "").Query; q != "endpoint_id="+hookID+"&status=delivered" {
		t.Fatalf("query %q", q)
	}
	_ = c.Webhooks.ListDeliveriesAutoPaging(nil)
}

func TestAnalytics(t *testing.T) {
	fs := newFixtureServer(t)
	fs.handle("GET", "/v1/analytics/campaigns/"+campID+"/daily", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"day":"2026-11-12","sent":10}]}`))
	})
	fs.handle("GET", "/v1/analytics/accounts", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"email_id":"` + mailboxID + `"}],"pagination":{"next_cursor":null,"has_more":false,"total":1}}`))
	})
	fs.handle("GET", "/v1/analytics/accounts/"+mailboxID, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	d, err := c.Analytics.Dashboard(ctx, &DashboardParams{Period: "30d", CampaignIDs: []string{"a", "b"}})
	must(t, err)
	if d.OverallStats.TotalEmailsSent != 420 || d.OverallStats.ReplyRate != 5.0 {
		t.Fatalf("dashboard %+v", d.OverallStats)
	}
	if q := expectCall(t, fs, "GET", "/analytics/dashboard", "").Query; q != "campaign_ids=a%2Cb&period=30d" {
		t.Fatalf("query %q", q)
	}

	ca, err := c.Analytics.Campaign(ctx, campID, &DateRangeParams{From: "2026-11-01", To: "2026-11-12"})
	must(t, err)
	if ca.Summary.EmailsSent != 180 || ca.DateRange.From.IsZero() {
		t.Fatalf("campaign analytics %+v", ca)
	}
	expectCall(t, fs, "GET", "/analytics/campaigns/"+campID, "")

	daily, err := c.Analytics.CampaignDaily(ctx, campID, nil)
	must(t, err)
	if len(daily) != 1 {
		t.Fatal("daily")
	}
	accts, err := c.Analytics.Accounts(ctx, &AccountAnalyticsParams{EmailIDs: []string{mailboxID}})
	must(t, err)
	if len(accts.Data) != 1 {
		t.Fatal("accounts")
	}
	walk, err := c.Analytics.AccountsAutoPaging(nil).Collect(ctx)
	must(t, err)
	if len(walk) != 1 {
		t.Fatal("accounts walk")
	}
	acct, err := c.Analytics.Account(ctx, mailboxID)
	must(t, err)
	if acct["status"] != "healthy" {
		t.Fatal("account")
	}
}

func TestSuppressions(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	page, err := c.Suppressions.List(ctx, &SuppressionListParams{Q: "example"})
	must(t, err)
	if len(page.Data) != 1 || page.Pagination.Next() != "" || page.Data[0].Kind != "email" {
		t.Fatalf("suppressions %+v", page)
	}
	expectCall(t, fs, "GET", "/suppressions", "")

	res, err := c.Suppressions.Add(ctx, &SuppressionAddParams{Entries: []SuppressionEntry{{Value: "example.org"}}, Reason: "Asked"})
	must(t, err)
	_ = res
	expectCall(t, fs, "POST", "/suppressions", `{"entries":[{"value":"example.org"}],"reason":"Asked"}`)

	must(t, c.Suppressions.Remove(ctx, "88888888-8888-4888-8888-111111111111"))
	expectCall(t, fs, "DELETE", "/suppressions/88888888-8888-4888-8888-111111111111", "")
}

func TestAITools(t *testing.T) {
	fs := newFixtureServer(t)
	c, _ := newTestClient(t, fs)
	ctx := context.Background()

	tools, err := c.AITools.List(ctx, &ToolListParams{Toolsets: []string{"campaigns", "inbox"}})
	must(t, err)
	if len(tools) != 1 || tools[0].Name != "list_campaigns" || tools[0].InputSchema == nil {
		t.Fatalf("tools %+v", tools)
	}
	if q := expectCall(t, fs, "GET", "/ai/tools", "").Query; q != "toolsets=campaigns%2Cinbox" {
		t.Fatalf("query %q", q)
	}

	res, err := c.AITools.Call(ctx, "list_campaigns", nil)
	must(t, err)
	if res.Name != "list_campaigns" || res.Untrusted || !strings.Contains(string(res.Result), "campaigns") {
		t.Fatalf("result %+v", res)
	}
	expectCall(t, fs, "POST", "/ai/tools/list_campaigns/call", `{}`)

	_, err = c.AITools.Call(ctx, "list_campaigns", map[string]any{"limit": 5})
	must(t, err)
	expectCall(t, fs, "POST", "/ai/tools/list_campaigns/call", `{"limit":5}`)
}
