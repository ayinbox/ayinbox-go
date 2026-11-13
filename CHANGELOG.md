# Changelog

All notable changes to this module are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [0.1.0] - unreleased

First release.

- `Client` with API key, base URL, timeout, retry and HTTP client options, reading `AYINBOX_API_KEY` and `AYINBOX_BASE_URL`
- The 62 tier 1 API methods: me, campaigns (with steps and leads), contacts, mailboxes, unibox, templates, webhooks, analytics, suppressions and AI tools
- `Request` and `Paginate` for any other route
- `*Error` with sentinels and predicates for every error category, including 429 answers without a `code`
- Retries on 429, 502, 503, 504 and connection errors, honouring `Retry-After`; keyless `POST` and `PATCH` are retried only on 429
- Per-call idempotency keys, timeouts, headers and request ids
- `Pager` with `Next`, `Collect` and, on Go 1.23 or later, `All`
- Webhook signature verification with a constant-time compare
