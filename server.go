package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"
)

type params struct {
	User     string `env:"PROXY_USER" envDefault:""`
	Password string `env:"PROXY_PASSWORD" envDefault:""`
	Port     string `env:"PROXY_PORT" envDefault:"1080"`
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %s", err)
	}

	// Read configuration into struct
	cfg := params{
		User:     getEnv("PROXY_USER", ""),
		Password: getEnv("PROXY_PASSWORD", ""),
		Port:     getEnv("PROXY_PORT", "1080"),
	}

	// Initialize socks5 config
	socks5Config := &proxy.Auth{
		User:     cfg.User,
		Password: cfg.Password,
	}
	dialer, err := proxy.SOCKS5("tcp", ":"+cfg.Port, socks5Config, proxy.Direct)
	if err != nil {
		log.Fatalf("Failed to create SOCKS5 proxy server: %s", err)
	}

	// Create a custom transport using the SOCKS5 proxy dialer
	tr := &http.Transport{
		Dial: dialer.Dial,
	}

	// Create an HTTP client using the custom transport
	client := &http.Client{
		Transport: tr,
	}

	log.Printf("Start listening proxy service on port %s", cfg.Port)

	// Start serving proxy requests
	err = http.ListenAndServe(":"+cfg.Port, limitConnections(authenticateRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Proxy the incoming request using the client
		resp, err := client.Do(r)
		if err != nil {
			log.Printf("Error while serving proxy request: %s", err)
			http.Error(w, "Proxy Error", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy the response headers and body to the client's response writer
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write([]byte{})
	}))), 100, 10)) // Set maximum 100 concurrent connections and allow 10 requests per second
	if err != nil {
		log.Fatalf("Error while serving proxy requests: %s", err)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// limitConnections wraps an HTTP handler with rate limiting middleware.
// It limits the number of concurrent connections and requests per second.
func limitConnections(next http.Handler, maxConnections, maxRequestsPerSec int) http.Handler {
	semaphore := make(chan struct{}, maxConnections)
	throttle := time.Tick(time.Second / time.Duration(maxRequestsPerSec))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			<-throttle
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		}
	})
}

// authenticateRequests wraps an HTTP handler with authentication middleware.
// It checks if the request is properly authenticated before allowing access.
func authenticateRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Implement authentication logic here
		// Example: Check if the request contains a valid access token
		if !isValidAccessToken(r.Header.Get("Authorization")) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isValidAccessToken checks if the provided access token is valid.
// Implement your own logic to validate the access token.
func isValidAccessToken(token string) bool {
	// Implement your access token validation logic here
	return true
}