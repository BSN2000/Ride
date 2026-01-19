package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyHeader = "Idempotency-Key"
	idempotencyTTL    = 24 * time.Hour
)

// cachedResponse stores the response for idempotent requests.
type cachedResponse struct {
	StatusCode int             `json:"status_code"`
	Body       json.RawMessage `json:"body"`
	Headers    http.Header     `json:"headers"`
}

// responseWriter wraps gin.ResponseWriter to capture the response.
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware returns middleware that handles idempotent requests.
func IdempotencyMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to mutating methods.
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		// Get idempotency key from header.
		key := c.GetHeader(idempotencyHeader)
		if key == "" {
			// No idempotency key - proceed normally.
			c.Next()
			return
		}

		ctx := c.Request.Context()
		cacheKey := "idempotency:" + key

		// Check for cached response.
		cached, err := getCachedResponse(ctx, redisClient, cacheKey)
		if err != nil && err != redis.Nil {
			// Redis error - proceed without idempotency.
			c.Next()
			return
		}

		if cached != nil {
			// Return cached response.
			for k, v := range cached.Headers {
				for _, val := range v {
					c.Header(k, val)
				}
			}
			c.Data(cached.StatusCode, "application/json", cached.Body)
			c.Abort()
			return
		}

		// Wrap response writer to capture response.
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = w

		// Process request.
		c.Next()

		// Cache the response.
		if c.Writer.Status() >= 200 && c.Writer.Status() < 500 {
			response := cachedResponse{
				StatusCode: c.Writer.Status(),
				Body:       w.body.Bytes(),
				Headers:    extractResponseHeaders(c),
			}
			_ = setCachedResponse(ctx, redisClient, cacheKey, &response, idempotencyTTL)
		}
	}
}

// getCachedResponse retrieves a cached response from Redis.
func getCachedResponse(ctx context.Context, client *redis.Client, key string) (*cachedResponse, error) {
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var cached cachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// setCachedResponse stores a response in Redis.
func setCachedResponse(ctx context.Context, client *redis.Client, key string, response *cachedResponse, ttl time.Duration) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return client.Set(ctx, key, data, ttl).Err()
}

// extractResponseHeaders extracts headers to cache.
func extractResponseHeaders(c *gin.Context) http.Header {
	headers := make(http.Header)
	// Only cache Content-Type header.
	if ct := c.Writer.Header().Get("Content-Type"); ct != "" {
		headers.Set("Content-Type", ct)
	}
	return headers
}
