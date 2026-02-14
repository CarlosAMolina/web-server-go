package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	dir := flag.String("dir", "", "Directory to serve static files from")
	certFile := flag.String("cert", "", "Path to the SSL certificate file")
	keyFile := flag.String("key", "", "Path to the SSL private key file")
	flag.Parse()
	if *dir == "" || *certFile == "" || *keyFile == "" {
		log.Fatal(`You must specify all the flags.
 Example: go run -dir content -cert /etc/letsencrypt/live/your-domain.com/fullchain.pem -key /etc/letsencrypt/live/your-domain.com/privkey.pem`)
	}
	fs := http.FileServer(http.Dir(*dir))
	http.Handle("/", http.StripPrefix("/", fs))
	port := ":8080"
	log.Println("Starting server at https://localhost" + port)
	err := http.ListenAndServeTLS(port, *certFile, *keyFile, nil)
	if err != nil {
		log.Fatalf("ListenAndServeTLS failed: %v", err)
	}
}
