# Security

Please report a vulnerability in this SDK by email to team@ayinbox.com. Do not open a public issue for it.

Include what you found, how to reproduce it and which version of `github.com/ayinbox/ayinbox-go` you used. We will confirm receipt and keep you informed while we fix it.

The SDK sends the API key only in the `Authorization` header and never includes it in errors or in the printed form of the client. If you find a path where it leaks, that is a vulnerability: report it as above.
