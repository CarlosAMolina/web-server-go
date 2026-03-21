# Copilot Instructions

## Commands

```bash
make build        # Cross-compile for linux/amd64 (CGO_ENABLED=0), outputs ./web-server
make run          # Run locally using config-test.json
make test         # Run all tests with verbose output
make format       # Run go fmt
make certs        # Generate self-signed TLS cert/key (server.cert, server.key)
make vulnerability # Check for known vulnerabilities
make modernize    # Apply Go modernization fixes
```

Run a single test:
```bash
go test -v -run TestRateLimiter
```

## Architecture

All server logic lives in `main.go`. The server has two modes controlled by `config.insecure`:

- **Insecure (HTTP)**: `loggingMiddleware` → `http.FileServer` — no security headers, no rate limiting
- **Secure (HTTPS)**: `loggingMiddleware` → `RateLimiter.Middleware` → `requestMiddleware` → `http.FileServer`

`requestMiddleware` does three things:
1. Redirects `wiki.<domain>` subdomain requests to `https://<domain>/wiki/index.html`
2. Rejects all non-GET methods with 405
3. Sets security headers: `Content-Security-Policy`, `Strict-Transport-Security`, `X-Content-Type-Options`, `X-Frame-Options`

Config is loaded from a JSON file passed via `-config <path>` at startup. `config.json` is the production template; `config-test.json` is used for local development.

## Key Conventions

- **Rate limiter burst**: `burstPerSecond = eventsPerSecond * 4`. This constant is duplicated in `NewRateLimiter` and `TestRateLimiter` — both must stay in sync.
- **Tests use TLS**: All tests spin up a TLS test server via `httptest.NewTLSServer` through the `startTestServer("content")` helper defined in `main_test.go`. Tests that need a custom server setup use `httptest.NewUnstartedServer` directly.
- **Test content**: Tests read from the real `content/` directory (e.g., `content/index.html`), so that directory must exist and be populated.
- **Build target**: The `make build` target always cross-compiles for `linux/amd64` with `CGO_ENABLED=0` to produce a static binary compatible with older glibc versions on VPS deployments.
