package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestRequestLogRedactsSensitiveQueryValues(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		want      string
		forbidden []string
	}{
		{
			name:      "oauth callback",
			query:     "code=private-code&state=private-state&page=2",
			want:      "/callback?code=%5BREDACTED%5D&page=2&state=%5BREDACTED%5D",
			forbidden: []string{"private-code", "private-state"},
		},
		{
			name:      "repeated and encoded keys",
			query:     "to%6ben=private-one&token=private-two&ACCESS_TOKEN=private-access&refresh_token=private-refresh",
			want:      "token=%5BREDACTED%5D&token=%5BREDACTED%5D",
			forbidden: []string{"private-one", "private-two", "private-access", "private-refresh"},
		},
		{
			name:      "malformed escape",
			query:     "code=private-code&state=%zz&token=private-token",
			want:      "/callback?[invalid query]",
			forbidden: []string{"private-code", "private-token", "%zz"},
		},
		{
			name:      "invalid separator",
			query:     "code=private-code;state=private-state&page=2",
			want:      "/callback?[invalid query]",
			forbidden: []string{"private-code", "private-state"},
		},
		{
			name:  "ordinary query",
			query: "page=2&search=office+supplies&category=food&category=travel",
			want:  "/callback?category=food&category=travel&page=2&search=office+supplies",
		},
		{
			name: "no query",
			want: "/callback ",
		},
	}

	// Configure the real request logger against a file so the assertions cover
	// what is emitted, rather than only a query sanitizing helper.
	capture, err := os.CreateTemp(t.TempDir(), "request-log")
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()

	originalStdout := os.Stdout
	os.Stdout = capture
	config := &settings.Config{
		EnableConsoleLog: true,
		EnableRequestLog: true,
		EnableQueryLog:   true,
		LogLevel:         settings.LOGLEVEL_INFO,
	}
	t.Cleanup(func() {
		os.Stdout = originalStdout
		if err := log.SetLoggerConfiguration(config, false); err != nil {
			t.Error(err)
		}
	})
	if err := log.SetLoggerConfiguration(config, false); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		RequestLog(core.WrapWebContext(c, nil))
	})
	router.GET("/callback", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startOffset, err := capture.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/callback", nil)
			request.URL.RawQuery = tt.query
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if request.URL.RawQuery != tt.query {
				t.Errorf("request query changed: got %q, want %q", request.URL.RawQuery, tt.query)
			}
			if response.Code != http.StatusNoContent {
				t.Errorf("unexpected response status: %d", response.Code)
			}

			// Read only this request's output. A missing log line must fail
			// without blocking the test suite.
			buffer := make([]byte, 4096)
			count, err := capture.ReadAt(buffer, startOffset)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			output := string(buffer[:count])
			if !strings.Contains(output, tt.want) {
				t.Errorf("log %q does not contain %q", output, tt.want)
			}
			for _, secret := range tt.forbidden {
				if strings.Contains(output, secret) {
					t.Errorf("log contains sensitive value %q", secret)
				}
			}
		})
	}
}
