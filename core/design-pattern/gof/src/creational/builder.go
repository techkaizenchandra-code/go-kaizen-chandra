package creational

import (
	"crypto/tls"
	"errors"
	"strconv"
	"time"
)

type CircuitBreaker struct {
	circuitState string
}

type RateLimiter struct {
	rateLimit int
}

type HTTPClient struct {
	baseURL        string
	timeout        time.Duration
	retryCount     int
	tlsConfig      *tls.Config
	enableTracing  bool
	enableMetrics  bool
	circuitBreaker CircuitBreaker
	rateLimiter    RateLimiter
}

type HTTPClientBuilder struct {
	client *HTTPClient
	err    error
}

func NewHTTPClientBuilder() *HTTPClientBuilder {
	return &HTTPClientBuilder{
		client: &HTTPClient{
			timeout:       5 * time.Second,
			retryCount:    3,
			enableTracing: false,
			enableMetrics: false,
		},
	}
}

func (b *HTTPClientBuilder) BaseURL(baseURL string) *HTTPClientBuilder {
	b.client.baseURL = baseURL
	return b
}

func (b *HTTPClientBuilder) Timeout(timeout time.Duration) *HTTPClientBuilder {
	b.client.timeout = timeout
	return b
}

func (b *HTTPClientBuilder) Retry(count int) *HTTPClientBuilder {
	b.client.retryCount = count
	return b
}
func (b *HTTPClientBuilder) TLSConfig(tlsConfig *tls.Config) *HTTPClientBuilder {
	b.client.tlsConfig = tlsConfig
	return b
}

func (b *HTTPClientBuilder) EnableTracing() *HTTPClientBuilder {
	b.client.enableTracing = true
	return b
}

func (b *HTTPClientBuilder) CircuitBreaker(circuitBreaker CircuitBreaker) *HTTPClientBuilder {
	b.client.circuitBreaker = circuitBreaker
	return b
}

func (b *HTTPClientBuilder) RateLimiter(rateLimiter RateLimiter) *HTTPClientBuilder {
	b.client.rateLimiter = rateLimiter
	return b
}

func (b *HTTPClientBuilder) EnableMetrics() *HTTPClientBuilder {
	b.client.enableMetrics = true
	return b
}

func (b *HTTPClientBuilder) Build() (*HTTPClient, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.client.baseURL == "" {
		return nil, errors.New("base URL is required")
	}
	if b.client.tlsConfig == nil {
		return nil, errors.New("TLS configuration is required")
	}
	return b.client, nil
}

func TestBuilder() {
	builder := NewHTTPClientBuilder()
	build, err := builder.
		BaseURL("https://example.com").
		TLSConfig(&tls.Config{}).
		EnableTracing().
		EnableMetrics().
		Timeout(10 * time.Second).
		Retry(5).
		CircuitBreaker(CircuitBreaker{circuitState: "closed"}).
		RateLimiter(RateLimiter{rateLimit: 10}).
		Build()
	if err != nil {
		panic(err)
	}
	println("baseURL " + build.baseURL)
	println("timeOut " + build.timeout.String())
	println("retryCount " + strconv.Itoa(build.retryCount))
	println("tracing " + strconv.FormatBool(build.enableTracing))
	println("matrics " + strconv.FormatBool(build.enableMetrics))
	println("circuitBreaker " + build.circuitBreaker.circuitState)
	println("rateLimiter " + strconv.Itoa(build.rateLimiter.rateLimit))
}
