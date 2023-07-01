package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"
)

// todo :
// 3. Security Improvements:
//    - Currently, the proxy credentials (user and password) are stored in environment variables.
//     It would be more secure to store them externally, such as in a configuration file or a secret management system.
//    - Consider implementing rate limiting to prevent abuse or DoS attacks on your proxy server.
//    - Apply proper authentication and authorization mechanisms to restrict access to authorized users only.

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
	err = http.ListenAndServe(":"+cfg.Port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
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
