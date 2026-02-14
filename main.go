package main

import (
	"flag"
	"log"
    "net/http"
)

func main() {
    certFile := flag.String("cert", "", "Path to the SSL certificate file")
    keyFile := flag.String("key", "", "Path to the SSL private key file")
    flag.Parse()
    if *certFile == "" || *keyFile == "" {
		log.Fatal(`You must specify both the certificate and key file paths.
 Example: go run -cert /etc/letsencrypt/live/your-domain.com/fullchain.pem -key /etc/letsencrypt/live/your-domain.com/privkey.pem`)
    }
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, HTTPS World with Let's Encrypt!"))
    })
	port := ":8080"
    log.Println("Starting server at https://localhost" + port)
    err := http.ListenAndServeTLS(port, *certFile, *keyFile, nil)
    if err != nil {
        log.Fatalf("ListenAndServeTLS failed: %v", err)
    }
}
