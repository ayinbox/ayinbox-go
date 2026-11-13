# Publishing

Go has no registry account, token or upload step. A module version exists once its tag is pushed to the public repository: `proxy.golang.org` fetches it on the first request and pkg.go.dev indexes it from the proxy. There is no trusted publisher to set up and no secret to store.

## One-time setup

1. Make `github.com/ayinbox/ayinbox-go` public. The module path `github.com/ayinbox/ayinbox-go` in `go.mod` must keep matching the repository path; the proxy cannot fetch a private repository.
2. Under Settings > Actions > General, allow GitHub Actions to create releases: "Workflow permissions" set to "Read and write permissions" (or keep "Read" and rely on the `permissions: contents: write` block in `release.yml`, which is enough when the organization does not restrict it).
3. Optional: protect `v*` tags under Settings > Rules so only maintainers can push them. A pushed version is permanent in the module proxy and its checksum database, so it can be retracted but never replaced.

## Releasing a version

1. Set `const Version` in `client.go` to the new version (it is the `User-Agent` and `release.yml` refuses a tag that does not match it), and move the `CHANGELOG.md` entry from "unreleased" to today's date. Commit and push to `main`, and wait for CI to pass.
2. Tag and push:

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. `release.yml` runs the tests, creates the GitHub release and requests the version from `proxy.golang.org`. pkg.go.dev shows it within minutes; to hurry it, open `https://pkg.go.dev/github.com/ayinbox/ayinbox-go@v0.1.0` and choose "Request".
4. Change the install section of `README.md` from `@main` to `go get github.com/ayinbox/ayinbox-go@latest`.

## Fixing a bad release

A published version cannot be deleted or re-tagged. Publish a new patch version with the fix, and add a `retract` directive for the bad one to `go.mod`:

```
retract v0.1.1 // Sends the wrong header; use v0.1.2.
```
