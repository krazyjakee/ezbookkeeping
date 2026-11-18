package middlewares

import (
	"net/url"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

// RequestLog logs the http request log
func RequestLog(c *core.WebContext) {
	start := time.Now()
	path := c.Request.URL.Path
	query := sanitizeRequestLogQuery(c.Request.URL.RawQuery)

	c.Next()

	now := time.Now()

	statusCode := c.Writer.Status()
	errorCode := int32(0)

	userId := "-"
	claims := c.GetTokenClaims()
	err := c.GetResponseError()

	clientIP := c.ClientIP()
	method := c.Request.Method

	if claims != nil {
		userId = utils.Int64ToString(claims.Uid)
	}

	if err != nil {
		errorCode = err.Code()
	}

	if query != "" {
		path = path + "?" + query
	}

	cost := now.Sub(start).Nanoseconds() / 1e6

	log.Requestf(c, "%d %d %s %s %s %s %dms", statusCode, errorCode, userId, clientIP, method, path, cost)
}

func sanitizeRequestLogQuery(rawQuery string) string {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		// A partially parsed query may omit sensitive fields. Never log the raw
		// input when decoding fails.
		return "[invalid query]"
	}

	for key, values := range query {
		switch strings.ToLower(key) {
		case "code", "state", "token", "access_token", "refresh_token":
			for i := range values {
				values[i] = "[REDACTED]"
			}
		}
	}

	return query.Encode()
}
