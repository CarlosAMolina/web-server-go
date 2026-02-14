# CGO_ENABLED = 0 to avoid these errors in the VPS when running the binary:
# ```
# /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.34' not found
# /lib/x86_64-linux-gnu/libc.so.6: version `GLIBC_2.32' not found
# ```
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o https .

certs:
	openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout server.key -out server.cert -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=CommonName"

format:
	go fmt

run:
	go run . -cert server.cert -key server.key -content content -logs /tmp
