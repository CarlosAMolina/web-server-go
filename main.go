package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/natefinch/lumberjack.v2"
)

const eventsPerSecond = 5
const burstPerSecond = eventsPerSecond * 4

// TODO. The rate limiter is initialized once and shared across all incoming requests.
// TODO. This global approach is not effective because it limits the total number of requests to the
// TODO. server, rather than preventing a single client from overwhelming it.
// TODO. To apply rate limits on a per-client basis, you would typically need to create and manage a
// TODO. collection of limiters, for example, in a map where each key is the client's IP address.
func rateLimitMiddleware(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(eventsPerSecond, burstPerSecond)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	configFile := flag.String("config", "", "Directory of the server configuration file")
	flag.Parse()
	if *configFile == "" {
		panic(`You must specify all the flags.
 Example: go run -config server.config`)
	}
	config := newConfig(configFile)
	logsFile := config.LogsDir + "/server.log"
	log.SetOutput(&lumberjack.Logger{
		Filename:   logsFile,
		MaxSize:    5,
		MaxBackups: 5,
		Compress:   true,
	})
	fs := http.FileServer(http.Dir(config.ContentDir))
	handler := loggingMiddleware(requestMiddleware(http.StripPrefix("/", fs)))
	handler = rateLimitMiddleware(handler)
	server := &http.Server{
		Addr:           config.Port,
		Handler:        handler,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    15 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
	fmt.Printf("Configuration: %+v\n", config)
	fmt.Println("Starting server at https://localhost" + config.Port)
	err := server.ListenAndServeTLS(config.CertFile, config.KeyFile)
	if err != nil {
		log.Fatalf("ListenAndServeTLS failed: %v", err)
	}
}

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Content-Security-Policy (CSP). Prevents Cross-Site Scripting (XSS) attacks.
		// Policy to only load resources (images, styles, scripts...) from the exact same origin as the
		// webpage itself. The browser will block inline scripts and scripts injected into attributes.
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		// Strict-Transport-Security (HSTS). Enforces the use of HTTPS, preventing man-in-the-middle attacks.
		// TODO. I couldn't check this header works, because if I turn off this header and I access
		// TODO. http instead of https, I get an error anyway.
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// X-Content-Type-Options. Prevents the browser from MIME-sniffing the content type of a response away
		// from the one declared by the server.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// X-Frame-Options. Protects against clickjacking attacks by controlling whether your site can be
		// embedded in elements like an <iframe>.
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}

func newConfig(configFile *string) Config {
	config := Config{}
	data, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}
	return config
}

type Config struct {
	CertFile   string `json:"cert"`
	ContentDir string `json:"content"`
	KeyFile    string `json:"key"`
	LogsDir    string `json:"logs"`
	Port       string `json:"port"`
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
