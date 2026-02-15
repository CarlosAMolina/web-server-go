# TODO

## Security best practices

1.  Implement Security Headers
* X-Content-Type-Options: Prevents the browser from MIME-sniffing the content type of a response away from the one declared by the server. The only valid value is nosniff.
* X-Frame-Options: Protects against clickjacking attacks by controlling whether your site can be embedded in an <iframe>. Set to DENY or SAMEORIGIN.
* X-XSS-Protection: A feature of Internet Explorer, Chrome, and Safari that stops pages from loading when they detect reflected cross-site scripting (XSS) attacks.

2.  Use a Robust Router (Mux)
While the default http.FileServer is fine for simple cases, using a dedicated router like gorilla/mux (https://github.com/gorilla/mux) or chi (https://github.com/go-chi/chi) provides more
control and can improve security. A good router will:
* Allow you to easily define which HTTP methods are allowed for each route (e.g., only GET for static assets).
* Provide a clear and maintainable way to structure your routes and handlers.

3.  Implement Rate Limiting
To protect against brute-force and denial-of-service (DoS) attacks, you should limit the number of requests a single client can make in a given time frame. The golang.org/x/time/rate
package provides an efficient token bucket-based rate limiter.

4.  Cross-Site Request Forgery (CSRF) Protection
If you add any forms or allow users to change state on the server, you must protect against CSRF attacks. This attack tricks a user's browser into making a request to your server that they
didn't intend to. Libraries like gorilla/csrf (https://github.com/gorilla/csrf) can help you implement this protection.

5.  Dependency Scanning
Your project's dependencies can be a source of vulnerabilities. You should regularly scan your dependencies for known security issues. You can use the official govulncheck tool from the Go team to do this:
1 go install golang.org/x/vuln/cmd/govulncheck@latest
2 govulncheck ./...
