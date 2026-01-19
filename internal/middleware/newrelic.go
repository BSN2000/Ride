package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// NewRelicMiddleware returns middleware that instruments requests with New Relic.
func NewRelicMiddleware(app *newrelic.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if app == nil {
			c.Next()
			return
		}

		txn := app.StartTransaction(c.Request.Method + " " + c.FullPath())
		defer txn.End()

		txn.SetWebRequestHTTP(c.Request)
		c.Set("newRelicTransaction", txn)

		writer := txn.SetWebResponse(c.Writer)
		c.Writer = &wrappedResponseWriter{
			ResponseWriter: c.Writer,
			writer:         writer,
		}

		c.Next()

		// Record error if present.
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				txn.NoticeError(err.Err)
			}
		}
	}
}

// wrappedResponseWriter wraps gin.ResponseWriter with New Relic's writer.
type wrappedResponseWriter struct {
	gin.ResponseWriter
	writer interface {
		WriteHeader(int)
	}
}

func (w *wrappedResponseWriter) WriteHeader(code int) {
	w.writer.WriteHeader(code)
	w.ResponseWriter.WriteHeader(code)
}
