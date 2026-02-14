package main

import (
	"flag"
	"log"
	"net/http"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	certFile := flag.String("cert", "", "Path to the SSL certificate file")
	contentDir := flag.String("content", "", "Directory to serve static files from")
	keyFile := flag.String("key", "", "Path to the SSL private key file")
	logsDir := flag.String("logs", "", "Directory to store logs")
	flag.Parse()
	if *certFile == "" || *contentDir == "" || *keyFile == "" || *logsDir == "" {
		panic(`You must specify all the flags.
 Example: go run -cert /etc/letsencrypt/live/your-domain.com/fullchain.pem -key /etc/letsencrypt/live/your-domain.com/privkey.pem -content content -logs /tmp`)
	}
	log.SetOutput(&lumberjack.Logger{
		Filename:   *logsDir + "/server.log",
		MaxSize:    10,
		MaxBackups: 3,
		Compress:   true,
	})
	fs := http.FileServer(http.Dir(*contentDir))
	http.Handle("/", loggingMiddleware(http.StripPrefix("/", fs)))
	port := ":8080"
	log.Println("Starting server at https://localhost" + port)
	err := http.ListenAndServeTLS(port, *certFile, *keyFile, nil)
	if err != nil {
		log.Fatalf("ListenAndServeTLS failed: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s \"%s %s %s\"",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			r.Proto,
		)
		var status int
		var size int64
		wrap := &responseWriter{ResponseWriter: w, status: &status, size: &size}
		next.ServeHTTP(wrap, r)
		log.Printf("%s \"%s %s %s\" %d %d",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			r.Proto,
			status,
			size,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status *int
	size   *int64
}

func (rw *responseWriter) WriteHeader(code int) {
	*rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	*rw.size += int64(n)
	return n, err
}
