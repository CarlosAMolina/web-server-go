package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/natefinch/lumberjack.v2"
)

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
	rl := NewRateLimiter(config.EventsPerSecond)
	handler = loggingMiddleware(rl.Middleware(handler))
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
		if after, ok := strings.CutPrefix(r.Host, "wiki."); ok {
			target := "https://" + after + "/wiki/index.html"
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
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

// TODO. Instead of apply global rate limit for all requests, apply per client IP.
type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(eventsPerSecond int) *RateLimiter {
	burstPerSecond := eventsPerSecond * 4 // A page loads: html, css and js -> minimum eventsPerSecond * 3
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(eventsPerSecond), burstPerSecond),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
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
	CertFile        string `json:"cert"`
	ContentDir      string `json:"content"`
	EventsPerSecond int    `json:"eventsPerSecond"` // I estimate 1 event per visitor.
	KeyFile         string `json:"key"`
	LogsDir         string `json:"logs"`
	Port            string `json:"port"`
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
