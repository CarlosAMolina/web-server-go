# TODO

## Security best practices

### Implement Rate Limiting
To protect against brute-force and denial-of-service (DoS) attacks, you should limit the number of requests a single client can make in a given time frame. The golang.org/x/time/rate
package provides an efficient token bucket-based rate limiter.

### Cross-Site Request Forgery (CSRF) Protection
If you add any forms or allow users to change state on the server, you must protect against CSRF attacks. This attack tricks a user's browser into making a request to your server that they
didn't intend to. Libraries like gorilla/csrf (https://github.com/gorilla/csrf) can help you implement this protection.

### Dependency Scanning
Your project's dependencies can be a source of vulnerabilities. You should regularly scan your dependencies for known security issues. You can use the official govulncheck tool from the Go team to do this:
1 go install golang.org/x/vuln/cmd/govulncheck@latest
2 govulncheck ./...

## Modify behaviour

- Redirect :80 request to :443.
- Manage subdomains: www and wiki.
